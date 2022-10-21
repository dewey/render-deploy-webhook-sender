# Render.com Deploy Webhook Sender

This is a simple service calling a webhook if there was a successful `deploy` event within a given window
on Render.com. This is a workaround for their [missing webhook support](https://feedback.render.com/features/p/deploy-webhooks). This
works well with [webhook-receiver](https://github.com/dewey/webhook-receiver).

# Deploy on fly.io

Set the secrets, the rest is already defined in the `fly.toml` file. Then run `flyctl deploy` to deploy the app.

```shell
  flyctl secrets set WEBHOOK_URL=https://example.com/hook
  flyctl secrets set API_TOKEN=rnd_redacted
  flyctl deploy
```
