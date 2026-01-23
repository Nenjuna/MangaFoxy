"""
Middleware for CSS/JS minification and compression
"""
import re
from django.utils.deprecation import MiddlewareMixin
from django.http import HttpResponse


class CSSMinifyMiddleware(MiddlewareMixin):
    """
    Minifies CSS in responses by removing whitespace and comments.
    Only applies to CSS files served through Django.
    """
    
    def process_response(self, request, response):
        # Only process CSS files
        if response.get('Content-Type', '').startswith('text/css'):
            content = response.content.decode('utf-8')
            
            # Remove comments
            content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
            
            # Remove extra whitespace
            content = re.sub(r'\s+', ' ', content)
            content = re.sub(r'\s*{\s*', '{', content)
            content = re.sub(r'\s*}\s*', '}', content)
            content = re.sub(r'\s*;\s*', ';', content)
            content = re.sub(r'\s*:\s*', ':', content)
            content = re.sub(r'\s*,\s*', ',', content)
            
            # Remove trailing semicolons before closing braces
            content = re.sub(r';}', '}', content)
            
            # Trim
            content = content.strip()
            
            response.content = content.encode('utf-8')
            response['Content-Length'] = str(len(response.content))
            
        return response
