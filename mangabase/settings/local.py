# mangabase/settings/local.py
from .base import *
import os

DEBUG = True

ALLOWED_HOSTS = [
    "localhost",
    "127.0.0.1",
    "0.0.0.0",
    "*",
    "192.168.0.3/16",  # LAN access
]

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": os.environ.get("DB_NAME", "mangabase"),
        "USER": os.environ.get("DB_USER", "mangabase_user"),
        "PASSWORD": os.environ.get("DB_PASS", "localpassword"),
        "HOST": os.environ.get("DB_HOST", "127.0.0.1"),
        "PORT": os.environ.get("DB_PORT", "5432"),
    }
}

STATIC_ROOT = BASE_DIR / "staticfiles"

