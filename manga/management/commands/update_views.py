from django.core.management.base import BaseCommand
from django.contrib.contenttypes.models import ContentType
from django.db.models import Count, F
from django.utils import timezone
from datetime import timedelta
from django.db import transaction

from manga.models import Manga, Chapter, ViewLog


class Command(BaseCommand):
    help = "Incrementally update Manga and Chapter view counts and clean up old ViewLogs."

    def add_arguments(self, parser):
        parser.add_argument(
            '--dry-run',
            action='store_true',
            help='Simulate the update without saving changes to the database.'
        )
        parser.add_argument(
            '--verbose',
            action='store_true',
            help='Print detailed output for each update.'
        )

    def handle(self, *args, **options):
        dry_run = options['dry_run']
        verbose = options['verbose']

        manga_type = ContentType.objects.get_for_model(Manga)
        chapter_type = ContentType.objects.get_for_model(Chapter)

        manga_logs = ViewLog.objects.filter(
            content_type=manga_type, processed=False)
        chapter_logs = ViewLog.objects.filter(
            content_type=chapter_type, processed=False)

        view_counts = manga_logs.values(
            'object_id').annotate(total=Count('id'))
        chapter_counts = chapter_logs.values(
            'object_id').annotate(total=Count('id'))

        self.stdout.write(
            f"Found {len(view_counts)} unprocessed Manga view groups.")
        self.stdout.write(
            f"Found {len(chapter_counts)} unprocessed Chapter view groups.")

        if dry_run:
            self.stdout.write(self.style.WARNING(
                "Dry run active — no updates will be saved."))

        with transaction.atomic():
            for entry in view_counts:
                if verbose or dry_run:
                    self.stdout.write(
                        f"Manga ID {entry['object_id']} => +{entry['total']} views")
                if not dry_run:
                    Manga.objects.filter(id=entry['object_id']).update(
                        view_count=F('view_count') + entry['total'])

            for entry in chapter_counts:
                if verbose or dry_run:
                    self.stdout.write(
                        f"Chapter ID {entry['object_id']} => +{entry['total']} views")
                if not dry_run:
                    Chapter.objects.filter(id=entry['object_id']).update(
                        view_count=F('view_count') + entry['total'])

            if not dry_run:
                manga_logs.update(processed=True)
                chapter_logs.update(processed=True)

        self.stdout.write(self.style.SUCCESS(
            "View counts updated." if not dry_run else "Simulated view count update."))

        # Cleanup processed logs older than 2 days
        cutoff_date = timezone.now() - timedelta(days=2)
        old_logs = ViewLog.objects.filter(timestamp__lt=cutoff_date)

        if verbose:
            self.stdout.write(
                f"Found {old_logs.count()} logs older than 2 days")
            self.stdout.write(f"Cutoff date: {cutoff_date}")

        if dry_run:
            self.stdout.write(self.style.WARNING(
                f"Dry run: Would delete {old_logs.count()} logs older than 2 days."))
        else:
            deleted_count, _ = old_logs.delete()
            self.stdout.write(self.style.SUCCESS(
                f"Deleted {deleted_count} old ViewLog entries."))
