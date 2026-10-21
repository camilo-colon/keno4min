module "subscriptions" {
  source = "./generated"

  event_bus_name = var.event_bus_name
  environment    = var.environment
}
