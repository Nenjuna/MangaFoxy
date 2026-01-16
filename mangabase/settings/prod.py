# mangabase/settings/prod.py
from .base import *
import os
import socket 

DEBUG = False

pod_ip = socket.gethostbyname(socket.gethostname())
ALLOWED_HOSTS = ["mangafoxy.com", 
    "django",
    "localhost", 
    "127.0.0.1",
    pod_ip]

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": os.environ.get("DB_NAME", "mangabase"),
        "USER": os.environ.get("DB_USER", "mangabase_user"),
        "PASSWORD": os.environ.get("DB_PASS"),
        "HOST": os.environ.get("DB_HOST", "postgres"),
        "PORT": os.environ.get("DB_PORT", "5432"),
    }
}

SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')
CSRF_TRUSTED_ORIGINS = ['https://mangafoxy.com']

# Make sure the STATIC_ROOT matches what WhiteNoise expects
STATIC_ROOT = BASE_DIR / "staticfiles"
