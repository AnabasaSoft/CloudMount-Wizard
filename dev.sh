#!/bin/bash

# Script de desarrollo rápido para CloudMount
# Compila y ejecuta en un solo comando

set -e

echo "🔨 Compilando CloudMount..."
go build -o /tmp/cloudmount-dev ./cmd/cloudmount

echo "🚀 Ejecutando..."
/tmp/cloudmount-dev "$@"
