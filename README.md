# Render.com Deploy Webhook Sender

This is a service calling a webhook if there was a successful deploy within a given window on Render.com. This is a
workaround for their [missing webhook support](https://feedback.render.com/features/p/deploy-webhooks).

# Usage

```shell
go run main.go --api-token=redacted --environment=development --webhook-url=http://localhost:12345/uid/hook
```