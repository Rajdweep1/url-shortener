#!/bin/bash

# CI/CD Setup Script
# Helps configure the CI/CD pipeline for first-time use

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 CI/CD Pipeline Setup${NC}"
echo "======================="

# Function to prompt for input
prompt_input() {
    local prompt="$1"
    local var_name="$2"
    local default="$3"
    
    if [[ -n "$default" ]]; then
        read -p "$prompt [$default]: " input
        eval "$var_name=\"${input:-$default}\""
    else
        read -p "$prompt: " input
        eval "$var_name=\"$input\""
    fi
}

# Function to update workflow files
update_workflows() {
    local old_image="$1"
    local new_image="$2"
    
    echo -e "${YELLOW}📝 Updating workflow files...${NC}"
    
    # Update all workflow files
    find .github/workflows -name "*.yml" -exec sed -i.bak "s|$old_image|$new_image|g" {} \;
    
    # Remove backup files
    find .github/workflows -name "*.bak" -delete
    
    echo -e "✅ ${GREEN}Updated Docker image name to: $new_image${NC}"
}

# Check if we're in the right directory
if [[ ! -d ".github/workflows" ]]; then
    echo -e "❌ ${RED}Error: .github/workflows directory not found${NC}"
    echo "Please run this script from the project root directory"
    exit 1
fi

echo -e "${YELLOW}📋 This script will help you configure the CI/CD pipeline${NC}"
echo ""

# Step 1: Docker Hub Configuration
echo -e "${BLUE}Step 1: Docker Hub Configuration${NC}"
echo "--------------------------------"

prompt_input "Enter your Docker Hub username" DOCKER_USERNAME
prompt_input "Enter your Docker image name" DOCKER_IMAGE "$DOCKER_USERNAME/url-shortener"

# Update workflow files with new image name
OLD_IMAGE="rajdweep1/url-shortener"
update_workflows "$OLD_IMAGE" "$DOCKER_IMAGE"

# Step 2: GitHub Secrets Setup
echo -e "\n${BLUE}Step 2: GitHub Secrets Setup${NC}"
echo "-----------------------------"

echo -e "${YELLOW}You need to add these secrets in GitHub:${NC}"
echo "Go to: GitHub Repository → Settings → Secrets and variables → Actions"
echo ""
echo -e "${GREEN}Required secrets:${NC}"
echo "• DOCKER_USERNAME: $DOCKER_USERNAME"
echo "• DOCKER_PASSWORD: Your Docker Hub access token"
echo ""
echo -e "${YELLOW}Optional secrets (for enhanced features):${NC}"
echo "• SNYK_TOKEN: For advanced security scanning"
echo "• CODECOV_TOKEN: For coverage reporting"
echo ""

read -p "Press Enter when you've added the GitHub secrets..."

# Step 3: Repository Settings
echo -e "\n${BLUE}Step 3: Repository Settings${NC}"
echo "---------------------------"

echo -e "${YELLOW}Recommended GitHub repository settings:${NC}"
echo ""
echo -e "${GREEN}Actions:${NC}"
echo "• Go to Settings → Actions → General"
echo "• Allow all actions and reusable workflows"
echo ""
echo -e "${GREEN}Security:${NC}"
echo "• Go to Settings → Code security and analysis"
echo "• Enable Dependency graph"
echo "• Enable Dependabot alerts"
echo "• Enable Dependabot security updates"
echo "• Enable CodeQL analysis"
echo ""
echo -e "${GREEN}Branch Protection:${NC}"
echo "• Go to Settings → Branches"
echo "• Add rule for 'main' branch"
echo "• Require status checks to pass"
echo "• Require branches to be up to date"
echo ""

read -p "Press Enter when you've configured the repository settings..."

# Step 4: Test Configuration
echo -e "\n${BLUE}Step 4: Test Configuration${NC}"
echo "--------------------------"

echo -e "${YELLOW}🧪 Testing CI/CD configuration...${NC}"

# Run validation script
if [[ -f "scripts/validate-cicd.sh" ]]; then
    ./scripts/validate-cicd.sh
else
    echo -e "⚠️  ${YELLOW}Validation script not found, skipping tests${NC}"
fi

# Step 5: Ready to Push
echo -e "\n${BLUE}Step 5: Ready to Deploy!${NC}"
echo "------------------------"

echo -e "${GREEN}🎉 CI/CD pipeline is configured!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Commit your changes:"
echo "   git add ."
echo "   git commit -m \"feat: configure CI/CD pipeline\""
echo ""
echo "2. Push to trigger the pipeline:"
echo "   git push origin main"
echo ""
echo "3. Watch the magic happen:"
echo "   • Go to GitHub → Actions tab"
echo "   • Watch your CI/CD pipeline run"
echo "   • Check the results and logs"
echo ""

echo -e "${BLUE}📊 What to expect:${NC}"
echo "• ✅ CI Pipeline: Will run immediately and should pass"
echo "• ✅ Security Scanning: Will start daily scans"
echo "• ✅ Docker Build: Will build and push images"
echo "• ✅ Dependabot: Will start monitoring dependencies"
echo ""

echo -e "${GREEN}🚀 Your URL shortener now has enterprise-grade CI/CD!${NC}"

# Create a summary file
cat > CICD_SETUP_SUMMARY.md << EOF
# CI/CD Setup Summary

## Configuration Applied
- **Docker Image**: $DOCKER_IMAGE
- **Setup Date**: $(date)
- **Configured By**: $(git config user.name) <$(git config user.email)>

## GitHub Secrets Required
- \`DOCKER_USERNAME\`: $DOCKER_USERNAME
- \`DOCKER_PASSWORD\`: Your Docker Hub access token

## Optional Secrets
- \`SNYK_TOKEN\`: For enhanced security scanning
- \`CODECOV_TOKEN\`: For coverage reporting

## Pipeline Status
- ✅ CI Pipeline: Ready
- ✅ Security Scanning: Ready  
- ✅ Docker Publishing: Ready (with secrets)
- ✅ Release Management: Ready

## Next Steps
1. Add GitHub secrets
2. Configure repository settings
3. Push code to trigger pipeline
4. Monitor Actions tab for results

## Support
- Documentation: docs/CICD.md
- Validation: ./scripts/validate-cicd.sh
- Setup: ./scripts/setup-cicd.sh
EOF

echo -e "\n📄 ${GREEN}Setup summary saved to: CICD_SETUP_SUMMARY.md${NC}"
