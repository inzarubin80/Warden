#!/usr/bin/env bash
set -euo pipefail

# Usage: generate_local_ca.sh [out_dir]
# Default out_dir: /tmp/warden_certs

OUT_DIR="${1:-/tmp/warden_certs}"
mkdir -p "$OUT_DIR"

CA_KEY="$OUT_DIR/warden_root_ca.key.pem"
CA_CERT="$OUT_DIR/warden_root_ca.crt.pem"
SERVER_KEY="$OUT_DIR/warden_dev_key.pem"
SERVER_CSR="$OUT_DIR/warden_dev.csr.pem"
SERVER_CERT="$OUT_DIR/warden_dev_cert.pem"
SAN_CONF="$OUT_DIR/warden_san.cnf"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required but not found. Install openssl and retry." >&2
  exit 2
fi

echo "Using output dir: $OUT_DIR"

if [ ! -f "$CA_KEY" ] || [ ! -f "$CA_CERT" ]; then
  echo "Generating root CA..."
  openssl genrsa -out "$CA_KEY" 4096
  openssl req -x509 -new -nodes -key "$CA_KEY" -sha256 -days 3650 -subj "/CN=Warden Local CA" -out "$CA_CERT"
else
  echo "Root CA already exists, skipping generation."
fi

echo "Generating server key and CSR..."
openssl genrsa -out "$SERVER_KEY" 2048

cat > "$SAN_CONF" <<EOF
[ req ]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[ req_distinguished_name ]
CN = localhost

[ v3_req ]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req -new -key "$SERVER_KEY" -out "$SERVER_CSR" -config "$SAN_CONF"

echo "Signing server certificate with root CA..."
openssl x509 -req -in "$SERVER_CSR" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial -out "$SERVER_CERT" -days 825 -sha256 -extensions v3_req -extfile "$SAN_CONF"

chmod 600 "$CA_KEY" "$SERVER_KEY"

# Also copy to the old /tmp expected paths for compatibility with main.go TLS_DEBUG behavior
cp -f "$SERVER_CERT" /tmp/warden_dev_cert.pem
cp -f "$SERVER_KEY" /tmp/warden_dev_key.pem
chmod 600 /tmp/warden_dev_key.pem

echo "Generated files:"
echo "  CA cert:   $CA_CERT"
echo "  CA key:    $CA_KEY"
echo "  Server cert: $SERVER_CERT (also copied to /tmp/warden_dev_cert.pem)"
echo "  Server key:  $SERVER_KEY (also copied to /tmp/warden_dev_key.pem)"

echo
echo "To trust the CA on Debian/Ubuntu (system-wide):"
echo "  sudo cp $CA_CERT /usr/local/share/ca-certificates/warden_root_ca.crt && sudo update-ca-certificates"
echo
echo "To trust the CA on macOS:"
echo "  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CA_CERT"
echo
echo "To trust the CA on Windows (PowerShell as admin):"
echo "  certutil -addstore -f \"ROOT\" \"$CA_CERT\""
echo
echo "Finished."


