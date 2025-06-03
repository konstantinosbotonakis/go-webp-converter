# Image to WebP Converter

A command-line tool written in Go to convert various image formats (JPEG, PNG, GIF) to WebP.

## Features

- Convert JPEG, PNG, and GIF images to WebP format.
- Process a single image file or recursively scan a directory for images.
- Content-based image type detection (not reliant on file extensions).
- Option to force overwrite existing output files.
- Cross-platform (builds for Windows, Linux, macOS).

## Prerequisites

- Go (version 1.20 or later recommended for building).

## Building from Source

1.  Clone the repository:
    ```bash
    git clone <repository-url> # Replace <repository-url> with the actual URL
    cd imageconverter
    ```
2.  Build the application:
    ```bash
    go build -o imageconverter cmd/imageconverter/main.go
    ```
    This will create an `imageconverter` executable in the current directory.

## Usage

The tool accepts a path to an image file or a directory and an optional `--force` flag.

```bash
./imageconverter --path <input_path> [--force]
```

**Arguments:**

-   `--path` (or `-p`): (Required) Path to the input image file or directory.
-   `--force` (or `-f`): (Optional) If set, allows overwriting existing `.webp` files. Defaults to `false`.

**Examples:**

-   **Convert a single image:**
    ```bash
    ./imageconverter --path /path/to/your/image.jpg
    ```
    _Output: `/path/to/your/image.webp`_

-   **Convert all images in a directory (recursively):**
    ```bash
    ./imageconverter --path /path/to/your/image_folder/
    ```

-   **Convert images in a directory and overwrite existing WebP files:**
    ```bash
    ./imageconverter --path /path/to/your/image_folder/ --force
    ```

## Supported Input Image Formats

The application detects image types based on their content. Currently supported input formats are:

-   JPEG
-   PNG
-   GIF (static GIFs)

## CI/CD

This project uses GitHub Actions for continuous integration and delivery:

-   **Linting:** `golangci-lint` is run on every push and pull request to ensure code quality.
-   **Testing:** Unit and integration tests (`go test ./...`) are automatically run.
-   **Releases:** When a new version tag (e.g., `v1.0.0`) is pushed to the `main` branch, a GitHub Release is automatically created with compiled binaries for Windows (amd64), Linux (amd64), macOS (amd64), and macOS (arm64).

<!-- Placeholder for CI badges if the repo is public on GitHub
[![Go Lint](<github-action-lint-url>)](<github-action-lint-url>)
[![Go Test](<github-action-test-url>)](<github-action-test-url>)
[![Go Release Build](<github-action-release-url>)](<github-action-release-url>)
-->

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the terms of the LICENSE file provided in this repository.
(Assuming a LICENSE file exists, which it does from the initial `ls()` output)
