# GitHub Pages Setup Instructions

This document explains how to enable GitHub Pages for this repository.

## Automatic Setup (Recommended)

The repository includes a GitHub Actions workflow that automatically builds and deploys the documentation when you push to the `main` branch.

### Steps to Enable:

1. **Go to Repository Settings**
   - Navigate to your repository on GitHub
   - Click on "Settings" tab

2. **Enable GitHub Pages**
   - Scroll down to "Pages" section in the left sidebar
   - Under "Source", select "GitHub Actions"
   - Save the changes

3. **Push to Main Branch**
   - The workflow will automatically trigger on push to `main` branch
   - Check the "Actions" tab to see the deployment progress

4. **Access Your Documentation**
   - After deployment, your docs will be available at:
   - `https://<username>.github.io/vmprober/`
   - Or if using custom domain: your configured domain

## Manual Setup (Alternative)

If you prefer to use GitHub Pages without Actions:

1. Go to Settings → Pages
2. Select "Deploy from a branch"
3. Choose branch: `main` or `gh-pages`
4. Select folder: `/docs` or `/ (root)`
5. Click Save

## Troubleshooting

### Workflow Not Running

- Check that GitHub Actions is enabled in repository settings
- Verify the workflow file is in `.github/workflows/pages.yml`
- Check the Actions tab for error messages

### Build Failures

- Ensure `Gemfile` exists in `docs/` directory
- Check that `_config.yml` is properly configured
- Review workflow logs in Actions tab

### Pages Not Updating

- Wait a few minutes for deployment to complete
- Check Actions tab for deployment status
- Clear browser cache if needed

## Custom Domain

To use a custom domain:

1. Add a `CNAME` file in `docs/` with your domain
2. Configure DNS records as per GitHub instructions
3. Enable "Enforce HTTPS" in Pages settings

## Local Testing

Test the documentation locally before pushing:

```bash
cd docs
bundle install
bundle exec jekyll serve
```

Then visit `http://localhost:4000` to preview.

