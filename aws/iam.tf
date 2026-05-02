resource "aws_iam_user" "immich_backup" {
  name = "immich-backup"
}

resource "aws_iam_policy" "immich_backup" {
  name        = "immich-backup-s3"
  description = "Allow Immich backup CronJobs to sync to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject",
      ]
      Resource = [
        aws_s3_bucket.immich_backup.arn,
        "${aws_s3_bucket.immich_backup.arn}/*",
      ]
    }]
  })
}

resource "aws_iam_user_policy_attachment" "immich_backup" {
  user       = aws_iam_user.immich_backup.name
  policy_arn = aws_iam_policy.immich_backup.arn
}

resource "aws_iam_access_key" "immich_backup" {
  user = aws_iam_user.immich_backup.name
}

# Seafile backup
resource "aws_iam_user" "seafile_backup" {
  name = "seafile-backup"
}

resource "aws_iam_policy" "seafile_backup" {
  name        = "seafile-backup-s3"
  description = "Allow Seafile backup CronJobs to sync to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject",
      ]
      Resource = [
        aws_s3_bucket.seafile_backup.arn,
        "${aws_s3_bucket.seafile_backup.arn}/*",
      ]
    }]
  })
}

resource "aws_iam_user_policy_attachment" "seafile_backup" {
  user       = aws_iam_user.seafile_backup.name
  policy_arn = aws_iam_policy.seafile_backup.arn
}

resource "aws_iam_access_key" "seafile_backup" {
  user = aws_iam_user.seafile_backup.name
}

# Health Hub backup
resource "aws_iam_user" "health_backup" {
  name = "health-backup"
}

resource "aws_iam_policy" "health_backup" {
  name        = "health-backup-s3"
  description = "Allow Health Hub backup CronJobs to sync to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject",
      ]
      Resource = [
        aws_s3_bucket.health_backup.arn,
        "${aws_s3_bucket.health_backup.arn}/*",
      ]
    }]
  })
}

resource "aws_iam_user_policy_attachment" "health_backup" {
  user       = aws_iam_user.health_backup.name
  policy_arn = aws_iam_policy.health_backup.arn
}

resource "aws_iam_access_key" "health_backup" {
  user = aws_iam_user.health_backup.name
}

# Read-only probe user — for ad-hoc cost/usage inspection across services.
# Uses AWS-managed ReadOnlyAccess so future diagnostics (Lambda, ECS, IAM
# audits, etc.) work without policy churn.
resource "aws_iam_user" "probe" {
  name = "homelab-probe"
}

resource "aws_iam_user_policy_attachment" "probe_readonly" {
  user       = aws_iam_user.probe.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}

resource "aws_iam_access_key" "probe" {
  user = aws_iam_user.probe.name
}
