#!/bin/bash

# CI/CD Validation Script
# Validates that all CI/CD components are properly configured

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 CI/CD Configuration Validation${NC}"
echo "=================================="

# Function to check if file exists
check_file() {
    local file=$1
    local description=$2
    
    if [[ -f "$file" ]]; then
        echo -e "✅ ${GREEN}$description${NC}: $file"
        return 0
    else
        echo -e "❌ ${RED}$description${NC}: $file (missing)"
        return 1
    fi
}

# Function to check if directory exists
check_directory() {
    local dir=$1
    local description=$2
    
    if [[ -d "$dir" ]]; then
        echo -e "✅ ${GREEN}$description${NC}: $dir"
        return 0
    else
        echo -e "❌ ${RED}$description${NC}: $dir (missing)"
        return 1
    fi
}

# Function to validate YAML syntax
validate_yaml() {
    local file=$1
    local description=$2
    
    if command -v yamllint >/dev/null 2>&1; then
        if yamllint "$file" >/dev/null 2>&1; then
            echo -e "✅ ${GREEN}$description YAML syntax${NC}: Valid"
            return 0
        else
            echo -e "❌ ${RED}$description YAML syntax${NC}: Invalid"
            return 1
        fi
    else
        echo -e "⚠️  ${YELLOW}$description YAML syntax${NC}: yamllint not installed, skipping"
        return 0
    fi
}

# Track validation results
TOTAL_CHECKS=0
PASSED_CHECKS=0

# Check GitHub Actions workflows
echo -e "\n${YELLOW}📋 GitHub Actions Workflows${NC}"
echo "----------------------------"

workflows=(
    ".github/workflows/ci.yml:CI Pipeline"
    ".github/workflows/cd.yml:CD Pipeline"
    ".github/workflows/security.yml:Security Pipeline"
    ".github/workflows/release.yml:Release Pipeline"
)

for workflow in "${workflows[@]}"; do
    IFS=':' read -r file desc <<< "$workflow"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if check_file "$file" "$desc"; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
        if validate_yaml "$file" "$desc"; then
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        fi
    fi
done

# Check configuration files
echo -e "\n${YELLOW}⚙️  Configuration Files${NC}"
echo "------------------------"

configs=(
    ".github/dependabot.yml:Dependabot Configuration"
    ".golangci.yml:GolangCI-Lint Configuration"
    "docker-compose.yml:Docker Compose"
    "Dockerfile:Docker Configuration"
)

for config in "${configs[@]}"; do
    IFS=':' read -r file desc <<< "$config"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if check_file "$file" "$desc"; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        if [[ "$file" == *.yml ]] || [[ "$file" == *.yaml ]]; then
            TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
            if validate_yaml "$file" "$desc"; then
                PASSED_CHECKS=$((PASSED_CHECKS + 1))
            fi
        fi
    fi
done

# Check documentation
echo -e "\n${YELLOW}📚 Documentation${NC}"
echo "------------------"

docs=(
    "docs/CICD.md:CI/CD Documentation"
    "README.md:Main Documentation"
)

for doc in "${docs[@]}"; do
    IFS=':' read -r file desc <<< "$doc"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if check_file "$file" "$desc"; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi
done

# Check directories
echo -e "\n${YELLOW}📁 Directory Structure${NC}"
echo "-----------------------"

directories=(
    ".github:GitHub Configuration"
    ".github/workflows:GitHub Actions Workflows"
    "docs:Documentation"
    "scripts:Build Scripts"
)

for directory in "${directories[@]}"; do
    IFS=':' read -r dir desc <<< "$directory"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if check_directory "$dir" "$desc"; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi
done

# Check for required tools
echo -e "\n${YELLOW}🔧 Required Tools${NC}"
echo "------------------"

tools=(
    "go:Go Language"
    "docker:Docker"
    "docker-compose:Docker Compose"
    "git:Git"
)

for tool in "${tools[@]}"; do
    IFS=':' read -r cmd desc <<< "$tool"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if command -v "$cmd" >/dev/null 2>&1; then
        version=$($cmd --version 2>/dev/null | head -n1 || echo "unknown")
        echo -e "✅ ${GREEN}$desc${NC}: $version"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        echo -e "❌ ${RED}$desc${NC}: Not installed"
    fi
done

# Check environment files
echo -e "\n${YELLOW}🌍 Environment Configuration${NC}"
echo "------------------------------"

env_files=(
    ".env.example:Environment Template"
)

for env_file in "${env_files[@]}"; do
    IFS=':' read -r file desc <<< "$env_file"
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if check_file "$file" "$desc"; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    fi
done

# Validate workflow triggers
echo -e "\n${YELLOW}🎯 Workflow Trigger Validation${NC}"
echo "--------------------------------"

TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if grep -q "on:" .github/workflows/ci.yml && grep -q "push:" .github/workflows/ci.yml; then
    echo -e "✅ ${GREEN}CI Pipeline Triggers${NC}: Configured"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "❌ ${RED}CI Pipeline Triggers${NC}: Missing or misconfigured"
fi

TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if grep -q "workflow_run:" .github/workflows/cd.yml; then
    echo -e "✅ ${GREEN}CD Pipeline Triggers${NC}: Configured"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "❌ ${RED}CD Pipeline Triggers${NC}: Missing or misconfigured"
fi

# Security checks
echo -e "\n${YELLOW}🔒 Security Configuration${NC}"
echo "----------------------------"

TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if grep -q "secrets\." .github/workflows/cd.yml; then
    echo -e "✅ ${GREEN}Secret Usage${NC}: Configured in CD pipeline"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "❌ ${RED}Secret Usage${NC}: No secrets configured"
fi

TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if grep -q "security" .github/workflows/security.yml; then
    echo -e "✅ ${GREEN}Security Scanning${NC}: Configured"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "❌ ${RED}Security Scanning${NC}: Not configured"
fi

# Final summary
echo -e "\n${BLUE}📊 Validation Summary${NC}"
echo "====================="

PASS_RATE=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))

echo -e "Total Checks: $TOTAL_CHECKS"
echo -e "Passed: ${GREEN}$PASSED_CHECKS${NC}"
echo -e "Failed: ${RED}$((TOTAL_CHECKS - PASSED_CHECKS))${NC}"
echo -e "Pass Rate: ${GREEN}$PASS_RATE%${NC}"

if [[ $PASS_RATE -ge 90 ]]; then
    echo -e "\n🎉 ${GREEN}CI/CD configuration is excellent!${NC}"
    exit 0
elif [[ $PASS_RATE -ge 75 ]]; then
    echo -e "\n✅ ${YELLOW}CI/CD configuration is good, minor improvements needed.${NC}"
    exit 0
else
    echo -e "\n❌ ${RED}CI/CD configuration needs significant improvements.${NC}"
    exit 1
fi
