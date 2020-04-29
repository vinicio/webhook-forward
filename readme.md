# Webhook Forward

Forward GitHub webhooks based on PRs branches and issues labels.

```
    b = the destination branch of PR
    l = the label of the issue    

    https://localhost:9090/webhook?
        b:master,l:team/backend=https://backend-webhook.com&
        b:frontend,l:team/frontend=https://frontend-webhook.com
```
