#!/bin/bash

# Windows Automation Assistant Build Script

echo "🔨 Building Windows Automation Assistant..."

# Build the assistant
go build -o assistant *.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "🚀 Run './assistant --help' to see available options"
    echo "📖 See README.md for documentation"
else
    echo "❌ Build failed!"
    exit 1
fi