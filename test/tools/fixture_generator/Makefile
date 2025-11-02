# Makefile for OpenZL C Fixture Generator
# This builds a test tool (NOT part of our Pure Go implementation)

# Paths
OPENZL_DIR = ../../../vendor/openzl
OPENZL_INCLUDE = $(OPENZL_DIR)/include
OPENZL_LIB = $(OPENZL_DIR)/lib

# Compiler settings
CC = gcc
CFLAGS = -Wall -Wextra -O2 -I$(OPENZL_INCLUDE)
LDFLAGS = -L$(OPENZL_LIB) -lopenzl -lzstd

# Output
TARGET = fixture_generator
OUTPUT_DIR = ../../fixtures

# Colors for output
GREEN = \033[0;32m
YELLOW = \033[0;33m
NC = \033[0m # No Color

.PHONY: all clean generate help

all: $(TARGET)

$(TARGET): main.c
	@echo "$(YELLOW)Building C fixture generator...$(NC)"
	$(CC) $(CFLAGS) -o $(TARGET) main.c $(LDFLAGS)
	@echo "$(GREEN)✓ Build complete: $(TARGET)$(NC)"

generate: $(TARGET)
	@echo ""
	@echo "$(YELLOW)Generating OpenZL test fixtures...$(NC)"
	@mkdir -p $(OUTPUT_DIR)/frames
	./$(TARGET) $(OUTPUT_DIR)
	@echo ""
	@echo "$(GREEN)✓ Fixtures ready for Pure Go testing!$(NC)"
	@echo ""

clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	rm -f $(TARGET)
	rm -f $(OUTPUT_DIR)/frames/*.bin
	@echo "$(GREEN)✓ Clean complete$(NC)"

help:
	@echo "OpenZL C Fixture Generator"
	@echo "=========================="
	@echo ""
	@echo "Targets:"
	@echo "  make         - Build the fixture generator"
	@echo "  make generate - Build and generate all test fixtures"
	@echo "  make clean   - Remove generated files"
	@echo "  make help    - Show this help"
	@echo ""
	@echo "Output:"
	@echo "  Fixtures are written to: $(OUTPUT_DIR)/frames/"
	@echo ""
	@echo "Usage:"
	@echo "  1. Run 'make generate' to create test fixtures"
	@echo "  2. Run 'go test ./internal/frame/...' to test Pure Go parser"
	@echo ""
	@echo "Note: This is a TEST TOOL only - not part of Pure Go implementation"
	@echo ""
