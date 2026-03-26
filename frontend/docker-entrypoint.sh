#!/bin/sh
# Dynamically read DNS resolver from /etc/resolv.conf and inject into nginx config
RESOLVER=$(grep 'nameserver' /etc/resolv.conf | head -1 | awk '{print $2}')
echo "Using DNS resolver: $RESOLVER"

cat > /etc/nginx/conf.d/default.conf << EOF
server {
    listen 3000;
    root /usr/share/nginx/html;
    index index.html;

    resolver ${RESOLVER} valid=5s ipv6=off;

    location /api {
        set \$upstream http://api:8080;
        proxy_pass \$upstream;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_read_timeout 60s;
    }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

exec nginx -g "daemon off;"
