# Strobe

Strobe is a controller that implements the periodic jobs defined in the
Lighthouse config ConfigMap:

```yaml
periodics:
- name: test
  cron: "@midnight"
```

This is done by watching the ConfigMap and adding each periodic job definition
as an item to a work queue which is processed by the controller's reconciliation
loop.
