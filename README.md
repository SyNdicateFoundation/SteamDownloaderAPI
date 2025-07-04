<h1 align="center">
<img src="https://i.imgur.com/kdyrovr.png" align="center" width="256">
</h1>


[![Go Report Card](https://goreportcard.com/badge/github.com/SyNdicateFoundation/SteamDownloaderAPI)](https://goreportcard.com/report/github.com/SyNdicateFoundation/SteamDownloaderAPI)
[![Go Build](https://github.com/SyNdicateFoundation/SteamDownloaderAPI/actions/workflows/go-build.yml/badge.svg)](https://github.com/SyNdicateFoundation/SteamDownloaderAPI/actions/workflows/go-build.yml)

A powerful Go-based reverse proxy for the Steam Workshop that dynamically injects download buttons into workshop and collection pages, allowing you to download items directly using a backend `steamcmd` instance.

***

## Overview

This project acts as a middleman between you and the Steam Community website. When you browse a Steam Workshop page through this proxy, it intelligently modifies the page on-the-fly to add "Download" buttons next to the standard "Subscribe" buttons. Clicking these new buttons triggers a backend process that uses `steamcmd` to download the workshop files to your server.

This is perfect for server administrators, content curators, or anyone who needs to obtain workshop files without subscribing to them through the Steam client.

***

## Features

-   **Reverse Proxy for Steam Workshop**: Forwards requests to `steamcommunity.com` seamlessly.
-   **Dynamic Content Injection**: Injects "Download" and "Download Collection" buttons directly into the HTML using `goquery`.
-   **URL Rewriting**: All asset URLs (CSS, JS, images) are rewritten to be served through the proxy, ensuring pages render correctly.
-   **`steamcmd` Integration**: Leverages a `steamcmd` wrapper to handle the actual download logic.
-   **Configurable Startup**: Use command-line flags to easily configure server settings.

***

## How It Works

1.  The user navigates to a proxied Steam Workshop URL (e.g., `http://localhost:8080/workshop/filedetails/?id=123456789`).
2.  The `SteamProxyHandler` receives the request and forwards it to the official Steam servers.
3.  Before sending the response back to the user, the `ModifyResponse` function intercepts it.
4.  The HTML body is parsed. The proxy finds all "Subscribe" buttons and injects new `<a>` tags next to them, pointing to the downloader API endpoints.
5.  All other URLs within the page (`href`, `src`, `srcset`) are rewritten to be relative, ensuring all subsequent requests for assets also go through the proxy.
6.  The modified, uncompressed HTML is sent to the user's browser.
7.  When the user clicks a "Download" button, a request is sent to an API endpoint like `/api/workshop/:app_id/:workshop_id`.
8.  The API handler calls the `steamcmd` wrapper, which executes the necessary commands (`workshop_download_item`) to download the files to the server.

***

## API Endpoints

-   `GET /api/workshop/:app_id/:workshop_id`
    -   Triggers a download for a single workshop item.
    -   **`app_id`**: The ID of the game (e.g., `4000` for Garry's Mod).
    -   **`workshop_id`**: The ID of the workshop file.

-   `GET /api/collection/:app_id/:collection_id`
    -   Triggers a download for all items within a collection.
    -   **`app_id`**: The ID of the game.
    -   **`collection_id`**: The ID of the workshop collection.

***

## Setup and Installation

### Prerequisites

-   [Go](https://golang.org/doc/install) (version 1.18 or newer)
-   A working installation of [`steamcmd`](https://developer.valvesoftware.com/wiki/SteamCMD) on the server where this API will run (or let the app install it for you).

### Steps

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/SyNdicateFoundation/SteamDownloaderAPI.git
    cd SteamDownloaderAPI
    ```

2.  **Install dependencies:**
    ```sh
    go mod tidy
    ```

3.  **Build the application:**
    ```sh
    go build -o steamdownloaderapi ./cmd/main.go
    ```

***

## Configuration & Usage

The application is configured at startup using command-line flags.

### Available Flags

-   `-listenhost`: The hostname or IP address for the server to listen on. (Default: `0.0.0.0`)
-   `-listenport`: The port for the server to listen on. (Default: `8080`)
-   `-steamcmdpath`: The directory path for `steamcmd`. (Default: `steamcmd`)
-   `-installsteamcmd`: If `true`, the application will install or update `steamcmd` on startup. (Default: `true`)
-   `-debug`: Enables debug mode for more verbose logging. (Default: `false`)
-   `-steamuser`: Your Steam username. Required for downloading certain content. (Default: `""`, will login as anonymous)
-   `-steampassword`: Your Steam password. (Default: `""`)

### Running the Server

Once built, you can run the application from your terminal.

**Basic startup (installs steamcmd to a 'steamcmd' folder and runs on port 8080):**
```sh
./steamdownloaderapi -steamuser your_username -steampassword your_password