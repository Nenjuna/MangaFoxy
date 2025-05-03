from django.contrib.contenttypes.models import ContentType
from manga.models import ViewLog

def get_client_ip(request):
    x_forwarded_for = request.META.get('HTTP_X_FORWARDED_FOR')
    if x_forwarded_for:
        return x_forwarded_for.split(',')[0]
    return request.META.get('REMOTE_ADDR')

def log_view(request, obj):
    ip = get_client_ip(request)
    session_id = request.session.session_key
    if not session_id:
        request.session.save()
        session_id = request.session.session_key

    content_type = ContentType.objects.get_for_model(obj)

    # if not ViewLog.objects.filter(
    #     content_type=content_type,
    #     object_id=obj.id,
    #     ip_address=ip,
    #     session_id=session_id,
    # ).exists():
    ViewLog.objects.create(
            content_type=content_type,
            object_id=obj.id,
            ip_address=ip,
            session_id=session_id,
        )
