# Notes
- Allowing only verified users works only for password flows
- Kratos will call the registration webhook before persisting the user, if the ldap search fails we will return an error and the user will be redirected to the registration_ui, there they will only see an error to contact the admin
- Kratos has pre-persist and post-persist registration webhooks. If `can_interrupt=True` OR `response.parse=True` then it is a pre-persist hook
- webhooks support authentication, we could use an opaque token to secure the API which will be shared using juju

The service that we will have to implement for verifying users using ldap will have to serve 2 URLs, one for responding to the webhook and one for serving the HTML page for the registration_ui

To try it, make sure you have the github client configuration in `.env` and run:
```
./start.sh
```

This will start the whole platform and a dummy client. Go to http://127.0.0.1:4446/ and try to log in with github.

If you inspect the id token that's minted, extra claims have been added.

To test denying user registration:
- go to `pkg/kratos/handlers.go` and uncomment the `hook` method on line 708 (and comment the other one).
- Make sure to remove your user from kratos:
```
$ kratos list identities -e http://localhost:4434
$ kratos delete identity -e http://localhost:4434 <your-UUID>
```
- Restart the login UI (simply re-run `start.sh`)

