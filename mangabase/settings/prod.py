# mangabase/settings/prod.py
from .base import *
import os

DEBUG = False

ALLOWED_HOSTS = ["mangafoxy.com"]

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

