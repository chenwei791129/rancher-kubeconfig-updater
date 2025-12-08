# Rancher Kubeconfig Updater

A command-line tool to update kubeconfig tokens for Rancher-managed Kubernetes clusters.

## Features

- Update kubeconfig tokens for all Rancher-managed clusters
- Auto-create kubeconfig entries for new clusters (optional)
- Backup kubeconfig before modifications

## Installation

```bash
git clone https://github.com/chenwei791129/rancher-kubeconfig-updater.git
cd rancher-kubeconfig-updater
go build
```

## Configuration

Set environment variables:

```bash
export RANCHER_URL=https://rancher.example.com
export RANCHER_USERNAME=your-username
export RANCHER_PASSWORD=your-password
```

Or use a `.env` file:

```bash
cp .env.example .env
# Edit .env with your credentials
```

## Usage

Update tokens for existing clusters:

```bash
./rancher-kubeconfig-updater
```

Auto-create entries for new clusters:

```bash
./rancher-kubeconfig-updater -a
```

## Options

- `-a`, `--auto-create` - Create kubeconfig entries for new clusters

## License

MIT License - see [LICENSE](LICENSE) file for details.
