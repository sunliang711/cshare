#!/bin/sh
if [ -n "$CROSSSHARE_SERVER" ]; then
  sed -i "s|window\.__CROSSSHARE_SERVER__|\"${CROSSSHARE_SERVER}\"|g" \
    /usr/share/nginx/html/app.js
fi
exec nginx -g 'daemon off;'
