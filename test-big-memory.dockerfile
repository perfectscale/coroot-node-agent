FROM alpine:latest

# Install a simple C compiler to create our memory-eating program
RUN apk add --no-cache gcc musl-dev

# Create a simple C program that allocates a lot of memory
RUN cat > /tmp/big_memory.c << 'EOF'
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

int main() {
    printf("Starting big memory test program...\\n");
    fflush(stdout);
    
    // Sleep for 2 seconds first to let the container start up
    sleep(2);
    
    printf("Starting memory allocation...\\n");
    fflush(stdout);
    
    // Allocate memory in large chunks to trigger different categories
    size_t chunk_size = 100 * 1024 * 1024; // 100MB chunks
    int chunks = 0;
    
    while (chunks < 6) { // Try to allocate 600MB total
        void *ptr = malloc(chunk_size);
        if (ptr == NULL) {
            printf("malloc failed after %d chunks\\n", chunks);
            break;
        }
        
        // Touch the memory to make sure it's actually allocated
        memset(ptr, 0, chunk_size);
        chunks++;
        printf("Allocated chunk %d (total: %d MB)\\n", chunks, chunks * 100);
        fflush(stdout);
        sleep(1);
    }
    
    printf("Finished allocating %d chunks\\n", chunks);
    fflush(stdout);
    
    // Keep the memory allocated for a bit
    sleep(5);
    
    return 0;
}
EOF

# Compile the program
RUN gcc -o /usr/local/bin/big_memory /tmp/big_memory.c

# Run the program
CMD ["/usr/local/bin/big_memory"] 