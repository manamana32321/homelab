resource "aws_s3_bucket" "immich_backup" {
  bucket = "immich-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "immich_backup" {
  bucket = aws_s3_bucket.immich_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }

}

resource "aws_s3_bucket_public_access_block" "immich_backup" {
  bucket = aws_s3_bucket.immich_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Seafile backup
resource "aws_s3_bucket" "seafile_backup" {
  bucket = "seafile-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "seafile_backup" {
  bucket = aws_s3_bucket.seafile_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }

}

resource "aws_s3_bucket_public_access_block" "seafile_backup" {
  bucket = aws_s3_bucket.seafile_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Health Hub backup
resource "aws_s3_bucket" "health_backup" {
  bucket = "health-backup-json-server"
}

resource "aws_s3_bucket_lifecycle_configuration" "health_backup" {
  bucket = aws_s3_bucket.health_backup.id

  # DB dumps: keep 30 days only
  rule {
    id     = "db-retention"
    status = "Enabled"

    filter {
      prefix = "db/"
    }

    expiration {
      days = 30
    }
  }
}

resource "aws_s3_bucket_public_access_block" "health_backup" {
  bucket = aws_s3_bucket.health_backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
