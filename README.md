# Render.com Deploy Webhook Sender

This is a simple service calling a webhook if there was a successful `deploy` event within a given window
on Render.com. This is a workaround for their [missing webhook support](https://feedback.render.com/features/p/deploy-webhooks). This
works well with [webhook-receiver](https://github.com/dewey/webhook-receiver).

# Usage

Set environment variables defined at the top of `main.go`, run the binary.