package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cronos/keno4min/internal/config"
	"github.com/cronos/keno4min/internal/users"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func newTestRepository(send httpClientFunc) *Repository {
	client := dynamodb.NewFromConfig(aws.Config{
		Region: "us-east-1", Credentials: aws.AnonymousCredentials{}, HTTPClient: send,
		Retryer: func() aws.Retryer { return aws.NopRetryer{} },
	})
	return NewDynamodbRepository(client, &config.Config{Dynamodb: config.DynamodbConfig{TableName: "keno-test"}})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/x-amz-json-1.0"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestUserStorageRoundTrip(t *testing.T) {
	user := &users.User{ID: uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897"), Username: "cajero", NodeId: uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")}
	wantItem := map[string]map[string]string{
		"PK": {"S": "USER#" + user.ID.String()}, "SK": {"S": "META"}, "username": {"S": "cajero"},
		"id": {"S": user.ID.String()}, "roleId": {"N": "0"}, "nodeId": {"S": user.NodeId.String()},
		"GSI1PK": {"S": "USER#cajero"}, "GSI1SK": {"S": "META"},
	}
	calls := 0
	repo := newTestRepository(func(req *http.Request) (*http.Response, error) {
		calls++
		var input struct {
			TableName, ConditionExpression string
			ConsistentRead                 bool
			Item, Key                      map[string]map[string]string
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.TableName != "keno-test" {
			t.Fatalf("wrong table: %s", input.TableName)
		}
		switch req.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.PutItem":
			if input.ConditionExpression != "attribute_not_exists(PK)" || !reflect.DeepEqual(input.Item, wantItem) {
				t.Fatalf("unexpected write: %+v", input)
			}
			return jsonResponse(http.StatusOK, `{}`), nil
		case "DynamoDB_20120810.GetItem":
			wantKey := map[string]map[string]string{"PK": {"S": "USER#" + user.ID.String()}, "SK": {"S": "META"}}
			if !input.ConsistentRead || !reflect.DeepEqual(input.Key, wantKey) {
				t.Fatalf("unexpected read: %+v", input)
			}
			body, err := json.Marshal(map[string]any{"Item": wantItem})
			if err != nil {
				t.Fatal(err)
			}
			return jsonResponse(http.StatusOK, string(body)), nil
		default:
			t.Fatalf("unexpected operation: %s", req.Header.Get("X-Amz-Target"))
			return nil, nil
		}
	})
	if err := repo.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindById(context.Background(), user.ID)
	if err != nil || !reflect.DeepEqual(got, user) || calls != 2 {
		t.Fatalf("got %+v, %v; calls=%d", got, err, calls)
	}
}

func TestSaveHandlesRedeliveryAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"duplicate", `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException"}`, false},
		{"missing table", `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			repo := newTestRepository(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(http.StatusBadRequest, tc.body), nil
			})
			err := repo.Save(context.Background(), &users.User{ID: uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897"), Username: "cajero", RoleId: 1, NodeId: uuid.NewV7()})
			if (err != nil) != tc.wantError || calls != 1 {
				t.Fatalf("error=%v, calls=%d", err, calls)
			}
			var missing *types.ResourceNotFoundException
			if tc.wantError && !errors.As(err, &missing) {
				t.Fatalf("lost original error: %v", err)
			}
		})
	}
}

func TestFindMissingOrInvalidUser(t *testing.T) {
	for _, tc := range []struct {
		body      string
		wantError bool
	}{
		{`{}`, false},
		{`{"Item":{}}`, false},
		{`{"Item":{"username":{"S":"cajero"},"nodeId":{"S":"f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6"}}}`, true},
		{`{"Item":{"id":{"S":"invalid"},"username":{"S":"cajero"},"nodeId":{"S":"f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6"}}}`, true},
		{`{"Item":{"id":{"S":"afed5b77-1e8a-471f-9398-489784a1b897"},"username":{"S":"cajero"},"nodeId":{"S":"invalid"}}}`, true},
		{`{"Item":{"nodeId":{"S":"f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6"}}}`, true},
		{`{"Item":{"username":{"S":"cajero"},"roleId":{"N":"-1"}}}`, true},
	} {
		repo := newTestRepository(func(*http.Request) (*http.Response, error) { return jsonResponse(http.StatusOK, tc.body), nil })
		got, err := repo.FindById(context.Background(), uuid.NewV7())
		if got != nil || (err != nil) != tc.wantError {
			t.Fatalf("body=%s: got %+v, %v", tc.body, got, err)
		}
	}
}

func TestRepositoryPreservesTransportErrors(t *testing.T) {
	failure := errors.New("connection failed")
	repo := newTestRepository(func(*http.Request) (*http.Response, error) { return nil, failure })
	if err := repo.Save(context.Background(), &users.User{ID: uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897"), Username: "cajero", RoleId: 1, NodeId: uuid.NewV7()}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := repo.FindById(context.Background(), uuid.NewV7()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestFindByUsernameQueriesIndex(t *testing.T) {
	id := uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897")
	nodeID := uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")
	item := `{"id":{"S":"` + id.String() + `"},"username":{"S":"Cajero"},"roleId":{"N":"2"},"nodeId":{"S":"` + nodeID.String() + `"}}`
	cursor := `{"PK":{"S":"USER#` + id.String() + `"},"SK":{"S":"META"},"GSI1PK":{"S":"USER#Cajero"},"GSI1SK":{"S":"META"}}`
	for _, tc := range []struct {
		name     string
		body     string
		wantUser bool
	}{
		{"found", `{"Items":[` + item + `]}`, true},
		{"missing", `{"Items":[]}`, false},
		{"first match only", `{"Items":[` + item + `],"LastEvaluatedKey":` + cursor + `}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			repo := newTestRepository(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("X-Amz-Target") != "DynamoDB_20120810.Query" {
					t.Fatal("expected Query")
				}
				var input struct {
					TableName, IndexName, KeyConditionExpression string
					ConsistentRead                               bool
					Limit                                        int
					ExpressionAttributeValues, ExclusiveStartKey map[string]map[string]string
				}
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatal(err)
				}
				wantValues := map[string]map[string]string{":pk": {"S": "USER#Cajero"}, ":sk": {"S": "META"}}
				if input.TableName != "keno-test" || input.IndexName != "GSI1" || input.ConsistentRead || input.Limit != 1 || input.KeyConditionExpression != "GSI1PK = :pk AND GSI1SK = :sk" || !reflect.DeepEqual(input.ExpressionAttributeValues, wantValues) {
					t.Fatalf("unexpected query: %+v", input)
				}

				calls++
				return jsonResponse(http.StatusOK, tc.body), nil
			})
			got, err := repo.FindByUsername(context.Background(), "Cajero")
			if err != nil || (got != nil) != tc.wantUser || calls != 1 {
				t.Fatalf("got %+v, %v, calls=%d", got, err, calls)
			}
			if tc.wantUser && !reflect.DeepEqual(got, &users.User{ID: id, Username: "Cajero", RoleId: 2, NodeId: nodeID}) {
				t.Fatalf("wrong user: %+v", got)
			}
		})
	}
}

func TestFindByUsernameRejectsInvalidItems(t *testing.T) {
	for _, item := range []string{
		`{"id":{"S":"invalid"},"username":{"S":"cajero"}}`,
		`{"id":{"S":"afed5b77-1e8a-471f-9398-489784a1b897"},"username":{"S":"cajero"},"nodeId":{"S":"invalid"}}`,
		`{"roleId":{"N":"-1"}}`,
	} {
		repo := newTestRepository(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"Items":[`+item+`]}`), nil
		})
		got, err := repo.FindByUsername(context.Background(), "cajero")
		if got != nil || err == nil || !strings.Contains(err.Error(), "cajero") {
			t.Fatalf("expected contextual decoding error: %+v, %v", got, err)
		}
	}
}

func TestFindByUsernamePreservesErrors(t *testing.T) {
	failure := errors.New("connection failed")
	repo := newTestRepository(func(*http.Request) (*http.Response, error) { return nil, failure })
	if _, err := repo.FindByUsername(context.Background(), "cajero"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	repo = newTestRepository(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException"}`), nil
	})
	var missing *types.ResourceNotFoundException
	if _, err := repo.FindByUsername(context.Background(), "cajero"); !errors.As(err, &missing) {
		t.Fatalf("lost DynamoDB error: %v", err)
	}
}
