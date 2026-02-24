#!/usr/bin/env bash
# Bootstrap script: creates IAM user + S3 state bucket for Terraform
# Run with AWS root/admin credentials: aws configure, then ./bootstrap.sh
set -euo pipefail

REGION="ap-northeast-2"
IAM_USER="homelab-terraform"
STATE_BUCKET="homelab-tfstate-361769566809"
POLICY_NAME="homelab-terraform-admin"

echo "=== 1. S3 state bucket ==="
if aws s3api head-bucket --bucket "$STATE_BUCKET" 2>/dev/null; then
  echo "Bucket $STATE_BUCKET already exists, skipping"
else
  aws s3api create-bucket \
    --bucket "$STATE_BUCKET" \
    --region "$REGION" \
    --create-bucket-configuration LocationConstraint="$REGION"
  aws s3api put-bucket-versioning \
    --bucket "$STATE_BUCKET" \
    --versioning-configuration Status=Enabled
  echo "Created $STATE_BUCKET with versioning enabled"
fi

echo ""
echo "=== 2. IAM user ==="
if aws iam get-user --user-name "$IAM_USER" 2>/dev/null; then
  echo "User $IAM_USER already exists, skipping"
else
  aws iam create-user --user-name "$IAM_USER"
  echo "Created user $IAM_USER"
fi

echo ""
echo "=== 3. IAM policy ==="
POLICY_DOC=$(cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "TerraformState",
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::homelab-tfstate-361769566809",
        "arn:aws:s3:::homelab-tfstate-361769566809/*"
      ]
    },
    {
      "Sid": "ManageIAM",
      "Effect": "Allow",
      "Action": [
        "iam:CreateUser", "iam:GetUser", "iam:DeleteUser",
        "iam:CreatePolicy", "iam:GetPolicy", "iam:DeletePolicy", "iam:GetPolicyVersion", "iam:ListPolicyVersions", "iam:CreatePolicyVersion", "iam:DeletePolicyVersion",
        "iam:AttachUserPolicy", "iam:DetachUserPolicy", "iam:ListAttachedUserPolicies",
        "iam:CreateAccessKey", "iam:DeleteAccessKey", "iam:ListAccessKeys",
        "iam:TagUser", "iam:TagPolicy"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ManageS3",
      "Effect": "Allow",
      "Action": [
        "s3:CreateBucket", "s3:DeleteBucket",
        "s3:GetBucketVersioning", "s3:PutBucketVersioning",
        "s3:GetBucketPolicy", "s3:PutBucketPolicy", "s3:DeleteBucketPolicy",
        "s3:GetLifecycleConfiguration", "s3:PutLifecycleConfiguration",
        "s3:GetBucketPublicAccessBlock", "s3:PutBucketPublicAccessBlock",
        "s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucket",
        "s3:GetBucketLocation", "s3:GetBucketTagging", "s3:PutBucketTagging",
        "s3:GetAccelerateConfiguration", "s3:GetBucketAcl",
        "s3:GetBucketCORS", "s3:GetBucketLogging", "s3:GetBucketObjectLockConfiguration",
        "s3:GetBucketRequestPayment", "s3:GetBucketWebsite",
        "s3:GetEncryptionConfiguration", "s3:GetReplicationConfiguration"
      ],
      "Resource": "*"
    }
  ]
}
EOF
)

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
POLICY_ARN="arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}"

if aws iam get-policy --policy-arn "$POLICY_ARN" 2>/dev/null; then
  echo "Policy $POLICY_NAME already exists, skipping"
else
  aws iam create-policy \
    --policy-name "$POLICY_NAME" \
    --policy-document "$POLICY_DOC"
  echo "Created policy $POLICY_NAME"
fi

aws iam attach-user-policy \
  --user-name "$IAM_USER" \
  --policy-arn "$POLICY_ARN"
echo "Attached $POLICY_NAME to $IAM_USER"

echo ""
echo "=== 4. Access Key ==="
KEY_OUTPUT=$(aws iam create-access-key --user-name "$IAM_USER")
ACCESS_KEY=$(echo "$KEY_OUTPUT" | jq -r '.AccessKey.AccessKeyId')
SECRET_KEY=$(echo "$KEY_OUTPUT" | jq -r '.AccessKey.SecretAccessKey')

echo ""
echo "========================================="
echo "Add to .envrc.local:"
echo "========================================="
echo "export AWS_ACCESS_KEY_ID=\"$ACCESS_KEY\""
echo "export AWS_SECRET_ACCESS_KEY=\"$SECRET_KEY\""
echo "========================================="
