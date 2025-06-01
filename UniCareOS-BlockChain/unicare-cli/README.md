# UniCareOS CLI

## Build

    go build -o unicare

## Usage

    ./unicare status        # Show node status (JSON summary)
    ./unicare health        # Show node health summary
    ./unicare liveness      # Check node liveness (true/false)
    ./unicare readiness     # Check node readiness (true/false)

    ./unicare status --output json