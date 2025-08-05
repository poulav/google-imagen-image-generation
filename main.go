package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "path"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "google.golang.org/genai"
)

var (
    s3Client     *s3.Client
    genaiClient  *genai.Client
    bucketName   string
    folderPrefix string
	region string
)

func init() {
	// load AWS Output Bucket Configuration
	region = os.Getenv("OUTPUT_BUCKET_REGION")
	if region == "" {
		region = "us-east-1" 
	}

    // Load AWS config & create S3 client
    awsCfg, err := config.LoadDefaultConfig(
		context.Background(), 
		config.WithRegion(region),
	)
    if err != nil {
        log.Fatalf("unable to load AWS SDK config: %v", err)
    }
    s3Client = s3.NewFromConfig(awsCfg)

    // Read bucket + optional folder prefix from env
    bucketName = os.Getenv("OUTPUT_BUCKET")
    if bucketName == "" {
        log.Fatalf("OUTPUT_BUCKET must be set")
    }
    folderPrefix = os.Getenv("OUTPUT_FOLDER") // e.g. "generated-images" or ""

    // Initialize GenAI client with API key from env
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        log.Fatalf("API_KEY must be set")
    }

	ctx := context.Background()
	genaiClient, err = genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:   apiKey,
		Backend:  genai.BackendGeminiAPI,
	})

    if err != nil {
        log.Fatalf("failed to create GenAI client: %v", err)
    }
}

type requestPayload struct {
    NumberOfImages   int32  `json:"numberOfImages"`             // optional, default 1
    AspectRatio      string `json:"aspectRatio,omitempty"`      // optional, default "SQUARE"
    PersonGeneration string `json:"personGeneration,omitempty"` // optional
    Prompt           string `json:"prompt"`                     // required
}

type responsePayload struct {
    ImageURLs []string `json:"imageUrls"`
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // 1) Parse and validate input
    var in requestPayload
    if err := json.Unmarshal([]byte(req.Body), &in); err != nil {
        return clientError(http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
    }
    if in.Prompt == "" {
        return clientError(http.StatusBadRequest, "prompt is required")
    }
    if in.NumberOfImages <= 0 {
        in.NumberOfImages = 1
    }
    if in.AspectRatio == "" {
        in.AspectRatio = "SQUARE"
    }

    // 2) Call Imagen 4
    genCfg := &genai.GenerateImagesConfig{
        NumberOfImages: in.NumberOfImages,
        AspectRatio:    in.AspectRatio,
    }
    if in.PersonGeneration != "" {
        genCfg.PersonGeneration = genai.PersonGeneration(in.PersonGeneration)
    }

    genResp, err := genaiClient.Models.GenerateImages(
        ctx,
        "imagen-4.0-generate-preview-06-06",
        in.Prompt,
        genCfg,
    )
    if err != nil {
        log.Printf("GenAI error: %v", err)
        return serverError(fmt.Sprintf("image generation failed: %v", err))
    }

    // 3) Upload each image directly from memory into S3
    var urls []string
    for idx, img := range genResp.GeneratedImages {
        key := path.Join(
            folderPrefix,
            fmt.Sprintf("imagen_%d_%s.png", idx, time.Now().Format("20060102T150405")),
        )
        _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
            Bucket:      aws.String(bucketName),
            Key:         aws.String(key),
            Body:        bytes.NewReader(img.Image.ImageBytes),
            ContentType: aws.String("image/png"),
        })
        if err != nil {
            log.Printf("S3 upload failed for %s: %v", key, err)
            return serverError(fmt.Sprintf("failed to upload image: %v", err))
        }

        // Construct a public URL (adjust region/domain if needed)
        url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, key)
        urls = append(urls, url)
    }

    // 4) Return JSON with all image URLs
    respBody, _ := json.Marshal(responsePayload{ImageURLs: urls})
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       string(respBody),
    }, nil
}

func clientError(status int, msg string) (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: status,
        Headers:    map[string]string{"Content-Type": "text/plain"},
        Body:       msg,
    }, nil
}

func serverError(msg string) (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusInternalServerError,
        Headers:    map[string]string{"Content-Type": "text/plain"},
        Body:       msg,
    }, nil
}

func main() {
    lambda.Start(handler)
}
