#!/usr/bin/env bash
set -euo pipefail

mv /tmp/recipe-rotation /usr/local/bin/recipe-rotation
chmod 755 /usr/local/bin/recipe-rotation
install -m 644 /tmp/recipe-rotation.service /etc/systemd/system/recipe-rotation.service
install -m 644 /tmp/nginx-recipe-rotation.conf /etc/nginx/sites-available/recipe-rotation
rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/recipe-rotation /etc/nginx/sites-enabled/recipe-rotation
install -d -o www-data -g www-data -m 0750 /var/lib/recipe-rotation
systemctl daemon-reload
systemctl enable recipe-rotation
# Replace binary on disk does not affect the running process; restart loads the new build.
systemctl restart recipe-rotation
systemctl is-active recipe-rotation
nginx -t
systemctl reload nginx
