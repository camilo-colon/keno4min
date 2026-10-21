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
	"github.com/cronos/keno4min/internal/nodes"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestRepository(send httpClientFunc) *Repository {
	client := dynamodb.NewFromConfig(aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  send,
		Retryer:     func() aws.Retryer { return aws.NopRetryer{} },
	})
	return NewDynamodbRepository(client, &config.Config{
		Dynamodb: config.DynamodbConfig{TableName: "keno-test"},
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/x-amz-json-1.0"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestSaveWritesNodeConditionally(t *testing.T) {
	id := uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")
	parentID := uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897")
	for _, parent := range []*uuid.UUID{nil, &parentID} {
		name := "root"
		if parent != nil {
			name = "child"
		}
		t.Run(name, func(t *testing.T) {
			var request struct {
				TableName           string
				ConditionExpression string
				Item                map[string]map[string]any
			}
			calls := 0
			repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
				calls++
				if target := req.Header.Get("X-Amz-Target"); target != "DynamoDB_20120810.PutItem" {
					t.Errorf("unexpected operation: %s", target)
				}
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				return jsonResponse(http.StatusOK, `{}`), nil
			})
			node := &nodes.Node{
				ID: id, ParentId: parent, Type: nodes.Cashier,
				Name: "Cajero Cali", Address: "Cali", TimeZone: "America/Bogota",
				Currency: "COP", BalanceCents: 0, Active: false,
			}
			if err := repository.Save(context.Background(), node); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || request.TableName != "keno-test" || request.ConditionExpression != "attribute_not_exists(PK)" {
				t.Fatalf("expected one conditional write to configured table: calls=%d request=%+v", calls, request)
			}
			want := map[string]map[string]any{
				"PK": {"S": "NODE#" + id.String()}, "SK": {"S": "META"},
				"id": {"S": id.String()}, "type": {"S": "Cashier"},
				"name": {"S": "Cajero Cali"}, "address": {"S": "Cali"},
				"timeZone": {"S": "America/Bogota"}, "currency": {"S": "COP"},
				"balanceCents": {"N": "0"}, "active": {"BOOL": false},
			}
			if parent != nil {
				want["parentId"] = map[string]any{"S": parent.String()}
				want["GSI1PK"] = map[string]any{"S": "NODE#" + parent.String()}
				want["GSI1SK"] = map[string]any{"S": "NODE#" + id.String()}
			}
			if !reflect.DeepEqual(request.Item, want) {
				t.Fatalf("unexpected item:\ngot  %#v\nwant %#v", request.Item, want)
			}
		})
	}
}

func TestSaveExistingNodeSucceedsWithoutAnotherWrite(t *testing.T) {
	calls := 0
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"already exists"}`), nil
	})
	if err := repository.Save(context.Background(), &nodes.Node{ID: uuid.NewV7()}); err != nil {
		t.Fatalf("existing node must be acknowledged: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no further write after condition failure, got %d calls", calls)
	}
}

func TestSavePreservesDynamoDBError(t *testing.T) {
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException","message":"table missing"}`), nil
	})
	id := uuid.NewV7()
	err := repository.Save(context.Background(), &nodes.Node{ID: id})
	var missingTable *types.ResourceNotFoundException
	if !errors.As(err, &missingTable) || !strings.Contains(err.Error(), id.String()) {
		t.Fatalf("expected original DynamoDB error with node identity: %v", err)
	}
}

func TestSavePreservesTransportError(t *testing.T) {
	failure := errors.New("connection failed")
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		return nil, failure
	})
	if err := repository.Save(context.Background(), &nodes.Node{ID: uuid.NewV7()}); !errors.Is(err, failure) {
		t.Fatalf("expected transport error to reach consumer: %v", err)
	}
}

func TestFindByIdReadsNode(t *testing.T) {
	id := uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")
	parentID := uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897")
	for _, parent := range []*uuid.UUID{nil, &parentID} {
		name := "root"
		if parent != nil {
			name = "child"
		}
		t.Run(name, func(t *testing.T) {
			item := map[string]map[string]any{
				"PK": {"S": "NODE#" + id.String()}, "SK": {"S": "META"},
				"id": {"S": id.String()}, "type": {"S": "Cashier"},
				"name": {"S": "Cajero Cali"}, "address": {"S": "Cali"},
				"timeZone": {"S": "America/Bogota"}, "currency": {"S": "COP"},
				"balanceCents": {"N": "12345"}, "active": {"BOOL": true},
			}
			if parent != nil {
				item["parentId"] = map[string]any{"S": parent.String()}
			}
			body, err := json.Marshal(map[string]any{"Item": item})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			repository := newTestRepository(func(req *http.Request) (*http.Response, error) {
				calls++
				if target := req.Header.Get("X-Amz-Target"); target != "DynamoDB_20120810.GetItem" {
					t.Errorf("unexpected operation: %s", target)
				}
				var request struct {
					TableName      string
					Key            map[string]map[string]string
					ConsistentRead bool
				}
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				wantKey := map[string]map[string]string{
					"PK": {"S": "NODE#" + id.String()}, "SK": {"S": "META"},
				}
				if request.TableName != "keno-test" || !request.ConsistentRead || !reflect.DeepEqual(request.Key, wantKey) {
					t.Fatalf("unexpected read: %+v", request)
				}
				return jsonResponse(http.StatusOK, string(body)), nil
			})
			got, err := repository.FindById(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			want := &nodes.Node{
				ID: id, ParentId: parent, Type: nodes.Cashier,
				Name: "Cajero Cali", Address: "Cali", TimeZone: "America/Bogota",
				Currency: "COP", BalanceCents: 12345, Active: true,
			}
			if calls != 1 || !reflect.DeepEqual(got, want) {
				t.Fatalf("expected one read returning %+v, got %+v (calls=%d)", want, got, calls)
			}
		})
	}
}

func TestFindByIdMissingNode(t *testing.T) {
	for _, body := range []string{`{}`, `{"Item":{}}`} {
		repository := newTestRepository(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, body), nil
		})
		node, err := repository.FindById(context.Background(), uuid.NewV7())
		if node != nil || err != nil {
			t.Fatalf("expected (nil, nil), got (%+v, %v)", node, err)
		}
	}
}

func TestFindByIdRejectsInvalidItem(t *testing.T) {
	id := uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")
	for _, tc := range []struct {
		name string
		body string
	}{
		{"invalid id", `{"Item":{"id":{"S":"invalid"}}}`},
		{"missing id", `{"Item":{"name":{"S":"Cajero"}}}`},
		{"invalid parent", `{"Item":{"id":{"S":"` + id.String() + `"},"parentId":{"S":"invalid"}}}`},
		{"invalid balance", `{"Item":{"id":{"S":"` + id.String() + `"},"balanceCents":{"N":"1.5"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := newTestRepository(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, tc.body), nil
			})
			node, err := repository.FindById(context.Background(), id)
			if node != nil || err == nil || !strings.Contains(err.Error(), id.String()) {
				t.Fatalf("expected decoding error with node identity, got (%+v, %v)", node, err)
			}
		})
	}
}

func TestFindByIdPreservesDynamoDBError(t *testing.T) {
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException","message":"table missing"}`), nil
	})
	id := uuid.NewV7()
	node, err := repository.FindById(context.Background(), id)
	var missingTable *types.ResourceNotFoundException
	if node != nil || !errors.As(err, &missingTable) || !strings.Contains(err.Error(), id.String()) {
		t.Fatalf("expected original DynamoDB error with node identity, got (%+v, %v)", node, err)
	}
}

func TestFindByIdPreservesTransportError(t *testing.T) {
	failure := errors.New("connection failed")
	repository := newTestRepository(func(*http.Request) (*http.Response, error) {
		return nil, failure
	})
	node, err := repository.FindById(context.Background(), uuid.NewV7())
	if node != nil || !errors.Is(err, failure) {
		t.Fatalf("expected original transport error, got (%+v, %v)", node, err)
	}
}
