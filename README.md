# wiki clipboard

this project has been on my mind for about 3 years. cross-platform copy-pasting of text / attachments has always been a pain.

the backend for this application is based on https://go.dev/doc/articles/wiki/, though I have extended the frontend and dockerised it to suit my needs

# usage

the deployed app is not for you, but i advocate for everyone to have access to such a wiki; as such this source code _is_ for you.

```bash
cd backend
chmod +x deploy.sh
./deploy.sh
```

# features

- qr-code for easy mobile navigation
- edit / view / delete / upload (attachment) endpoints
- persistence. saves txt files and attachments + reloads them on docker restarts.