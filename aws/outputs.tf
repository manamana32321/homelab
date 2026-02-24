output "immich_backup_access_key_id" {
  description = "Access Key ID for immich-backup IAM user"
  value       = aws_iam_access_key.immich_backup.id
}

output "immich_backup_secret_access_key" {
  description = "Secret Access Key for immich-backup IAM user"
  value       = aws_iam_access_key.immich_backup.secret
  sensitive   = true
}

output "immich_backup_bucket" {
  description = "S3 bucket name for Immich backups"
  value       = aws_s3_bucket.immich_backup.bucket
}
