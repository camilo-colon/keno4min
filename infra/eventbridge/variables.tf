variable "aws_region" {
  description = "AWS region of the existing EventBridge bus and the queues."
  type        = string
}

variable "event_bus_name" {
  description = "Name of the existing EventBridge bus in the same AWS account and region."
  type        = string
}

variable "environment" {
  description = "Deployment environment, such as dev, staging, or prod."
  type        = string
}
