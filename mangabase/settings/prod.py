# mangabase/settings/prod.py
from .base import *
import os

DEBUG = False
ALLOWED_HOSTS = ['mangafoxy.com']

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.mysql',
        'NAME': 'manglevi_django-test',
        'USER': 'manglevi_django',
        'PASSWORD': os.environ.get('DB_PASS'),
        'HOST': 'localhost',
        'PORT': '3306',
    }
}
