# Webhook Forward

Forward GitHub webhooks based on PRs branches and issues labels.

```
    b = the destination branch of PR
    l = the label of the issue    

    http://localhost:9090/webhook?
        b:master,l:team/backend=https://discordapp.com/api/webhooks/6902/R_Na/github&
        b:frontend,l:team/frontend=https://discordapp.com/api/webhooks/677/-I2S/github
```
