#!/bin/bash
set -e

# Run migrations (only once)
echo "Running migrations..."
python manage.py migrate --noinput

# Start Gunicorn
echo "Starting Gunicorn..."
exec gunicorn mangabase.wsgi:application --bind 0.0.0.0:8000 --workers 3