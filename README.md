# Grafana data source plugin for Harper

## Getting started

Requires Grafana 7.0 or higher.

Clone this repo into your `grafana/plugins` directory.

This plugin is not yet signed, so you'll need to [configure Grafana to load it anyway](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/#allow_loading_unsigned_plugins).

## Configuration

When you create a new Grafana datasource with this plugin,
you'll need to give it the full URL (including port) to
your Harper Operations API endpoint.

Then you'll need to give Grafana the username and password
you'd like to use to access your Harper instance.

## Development

### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins/grafana-plugin-sdk-for-go) dependency to the latest minor version:

   ```bash
   go get -u github.com/grafana/grafana-plugin-sdk-go
   go mod tidy
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   mage -v
   ```

3. List all available Mage targets for additional commands:

   ```bash
   mage -l
   ```

### Frontend

1. Install dependencies

   ```bash
   npm install
   ```

2. Build plugin in development mode and run in watch mode

   ```bash
   npm run dev
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes, requires git init first
   npm run test

   # Exits after running all the tests
   npm run test:ci
   ```

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Cypress)

   ```bash
   # Spins up a Grafana instance first that we tests against
   npm run server

   # Starts the tests
   npm run e2e
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```

### Dev workflow

1. Run `npm run dev` in one terminal
   - This will watch your frontend code for changes
1. Build the backend: `mage -v build:linuxARM64`
    - Because this will run in a Docker container, you should always build it for Linux
    - You will have to manually rerun this when the backend code changes
1. Run `docker compose up` in another terminal
    - This will need to be restarted when the backend is rebuilt
1. Access Grafana at http://localhost:3000/
