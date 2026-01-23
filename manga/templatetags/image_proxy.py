from django import template
from django.urls import reverse
from urllib.parse import quote

register = template.Library()


@register.filter
def proxy_image(image_url):
    """
    Template filter to convert external image URLs to proxied URLs.
    Usage: {{ image_url|proxy_image }}
    """
    if not image_url:
        return image_url
    
    # Don't proxy local/static images
    if image_url.startswith('/') or image_url.startswith('data:') or image_url.startswith('http://localhost') or image_url.startswith('https://localhost'):
        return image_url
    
    # Encode the URL for the proxy
    encoded_url = quote(image_url, safe='')
    return reverse('image_proxy') + f'?url={encoded_url}'
