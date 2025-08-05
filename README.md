# Generate Imagen Function

This repository provides an AWS Lambda function written in Go that leverages Google Imagen 4 (Gemini API) to generate images and store them in an S3 bucket. It includes:

- **CloudFormation template** (`infrastructure.yaml`): Deploys the Lambda function, IAM role, and a public Function URL.
- **Lambda handler** (`main.go`): Accepts JSON requests, calls the Google GenAI API, and uploads generated images to S3.

---

## Prerequisites

- **AWS CLI** installed and configured with permissions to create IAM roles, Lambda functions, and S3 buckets.
- **Go (>= 1.18)** installed locally.
- **S3 bucket** for uploading Lambda deployment packages and for storing output images.
- **Google Cloud GenAI (Gemini) API key** with image-generation access.

---

## Repository Structure

```text
├── main.go            # Lambda function code
├── infrastructure.yaml      # CloudFormation template
└── README.md          # This documentation
```

---

## Build and Package the Lambda

1. **Clone the repository**

   ```bash
   git clone <your-repo-url>
   cd <repo-folder>
   ```

2. **Build the Go binary for Linux**

   ```bash
   GOOS=linux GOARCH=amd64 go build -o main main.go
   ```

3. **Create a ZIP package**

   ```bash
   zip function.zip main
   ```

4. **Upload to S3**

   ```bash
   aws s3 cp function.zip s3://<CodeS3Bucket>/<CodeS3Key>
   ```

---

## Deploy with CloudFormation

Use the AWS CLI to deploy the CloudFormation stack:

```bash
aws cloudformation deploy \
  --template-file template.yaml \
  --stack-name Imagen4Stack \
  --parameter-overrides \
      CodeS3Bucket=<CodeS3Bucket> \
      CodeS3Key=<CodeS3Key> \
      GeminiAPIKey=<YOUR_GEMINI_API_KEY> \
      GeminiOutputBucket=<OutputBucketName> \
      GeminiOutputFolder=<OutputFolder> \
  --capabilities CAPABILITY_NAMED_IAM
```

- **CodeS3Bucket**: S3 bucket with the Lambda ZIP file.
- **CodeS3Key**: Path/key for `function.zip`.
- **GeminiAPIKey**: Your Google Imagen API key.
- **GeminiOutputBucket**: Public S3 bucket for generated images.
- **GeminiOutputFolder**: (Optional) Prefix/folder in the output bucket.

Once deployment completes, CloudFormation outputs:

- `LambdaFunctionArn` — ARN of the Lambda function.
- `LambdaExecutionRoleArn` — ARN of the IAM role.
- `FunctionInvokeUrl` — The public URL to invoke the Lambda.

---

## Invoke the Function

Use `curl` or any HTTP client to call the Function URL:

```bash
curl -X POST <FunctionInvokeUrl> \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A serene mountain lake at sunrise",
    "numberOfImages": 1,
    "aspectRatio": "SQUARE"
  }'
```

**Response**:

```json
{
  "imageUrls": [
    "https://<YourBucket>.s3.<region>.amazonaws.com/<OutputFolder>/imagen_0_20250805T123456.png"
  ]
}
```

---

## Environment Variables

The Lambda function reads these environment variables:

- `OUTPUT_BUCKET` — Name of the S3 bucket for images.
- `OUTPUT_FOLDER` — (Optional) S3 prefix for storing images.
- `API_KEY` — Google Gemini API key.
- `OUTPUT_BUCKET_REGION` — AWS region of the output bucket (default `us-east-1`).

These are set automatically by the CloudFormation template.

---

## Cleanup

To remove all resources created by this stack:

```bash
aws cloudformation delete-stack --stack-name Imagen4Stack
```

### Reference
[Google Imagen] (https://ai.google.dev/gemini-api/docs/imagen#go)
