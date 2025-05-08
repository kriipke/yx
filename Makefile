# Define variables
BINDIR := binaries
YQ_URL := https://github.com/mikefarah/yq/releases/download/v4.45.2/yq_linux_amd64
DELTA_URL := https://github.com/dandavison/delta/releases/download/0.18.2/delta-0.18.2-x86_64-unknown-linux-gnu.tar.gz
DELTA_DIR := delta-0.18.2-x86_64-unknown-linux-gnu
OUTPUT := yiff

# Default target
all: $(OUTPUT)

# Create binaries directory
$(BINDIR):
	mkdir -p $(BINDIR)

# Download yq binary
$(BINDIR)/yq: $(BINDIR)
	curl -sSLo $@ $(YQ_URL) && chmod a+x $@

# Download and extract delta binary
$(BINDIR)/delta: $(BINDIR)
	curl -sSL $(DELTA_URL) | tar -C $(BINDIR) --strip=1 -xzvf - $(DELTA_DIR)/delta

# Build the Go project
$(OUTPUT): $(BINDIR)/yq $(BINDIR)/delta
	go build -o $(OUTPUT)

# Clean up generated files
clean:
	rm -rf $(BINDIR) $(OUTPUT)

# Phony targets
.PHONY: all clean
