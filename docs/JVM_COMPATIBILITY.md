# JVM Compatibility Guide

This document describes the JVM compatibility features and limitations of the coroot-node-agent's JVM monitoring capabilities.

## Overview

The coroot-node-agent monitors JVM applications by:
1. Using jattach to connect to running JVMs via Unix sockets
2. Executing jcmd commands (VM.flags, VM.system_properties, VM.version)
3. Parsing the output to extract heap configuration and garbage collector information
4. Falling back to command-line argument parsing when jcmd fails

## Supported JVM Implementations

### ✅ Fully Supported

#### Oracle HotSpot / OpenJDK HotSpot
- **Versions**: Java 8+
- **Detection**: VM flags contain "hotspot", "openjdk", or "oracle"
- **Heap Flags**: `-XX:MaxHeapSize`, `-XX:InitialHeapSize`, `-XX:MaxRAMPercentage`, `-XX:InitialRAMPercentage`
- **GC Types**: G1GC, ParallelGC, SerialGC, ZGC, ShenandoahGC, ConcMarkSweepGC
- **Fallback**: Command line parsing for `-Xmx`, `-Xms` flags

#### GraalVM
- **Versions**: GraalVM 19+
- **Detection**: VM flags contain "graalvm" or "graal"
- **Compatibility**: Uses HotSpot-compatible flag parsing
- **Notes**: Inherits HotSpot behavior with potential GraalVM-specific extensions

### ⚠️ Limited Support

#### Eclipse OpenJ9 / IBM J9
- **Versions**: OpenJ9 0.8+, IBM J9 8+
- **Detection**: VM flags/properties contain "openj9", "eclipse", or "ibm"
- **Heap Flags**: `-Xmx`, `-Xms` (different from HotSpot)
- **GC Types**: Gencon, Optthruput, Optavgpause, Balanced
- **Limitations**: 
  - Different flag naming conventions
  - VM.flags command may not be available or return different format
  - Percentage-based heap sizing not supported
- **Fallback**: Enhanced command-line parsing for OpenJ9-specific flags

### ❌ Unsupported

#### Very Old Java Versions (< Java 8)
- **Issue**: Limited or no jcmd support
- **Workaround**: Command-line fallback parsing only
- **Limitations**: Missing modern GC options, percentage-based heap sizing

#### Custom JVM Implementations
- **Examples**: Azul Zing, Excelsior JET, etc.
- **Issue**: Unknown flag formats and jcmd compatibility
- **Workaround**: Basic command-line parsing may work

## Compatibility Matrix

| JVM Implementation | Version | Year | VM.flags | System Properties | GC Detection | Heap Detection | Status |
|-------------------|---------|------|----------|-------------------|--------------|----------------|--------|
| Oracle HotSpot    | 8+      | 2014+ | ✅       | ✅                | ✅           | ✅             | Full   |
| OpenJDK HotSpot   | 8+      | 2014+ | ✅       | ✅                | ✅           | ✅             | Full   |
| GraalVM           | 19+     | 2019+ | ✅       | ✅                | ✅           | ✅             | Full   |
| Eclipse OpenJ9    | 0.8+    | 2017+ | ⚠️       | ✅                | ⚠️           | ⚠️             | Limited|
| IBM J9            | Any     | ~2000 | ⚠️       | ✅                | ⚠️           | ⚠️             | Limited|
| Oracle HotSpot    | 6-7     | 2006-2011 | ❌       | ❌                | ⚠️           | ⚠️             | Minimal|
| Azul Zing         | Any     | 2010+ | ❌       | ❌                | ❌           | ❌             | None   |

## Feature Detection

### JVM Vendor Detection Process

1. **System Properties Check** (Most Accurate)
   - `java.vm.name` property
   - `java.vendor` property

2. **Version Information Check**
   - VM.version command output

3. **VM Flags Pattern Matching** (Fallback)
   - Search for vendor-specific strings in flags output

4. **Command Line Analysis** (Last Resort)
   - Parse JVM arguments from process command line

### Parsing Strategy by Vendor

#### HotSpot/OpenJDK
```
-XX:MaxHeapSize=2147483648
-XX:InitialHeapSize=268435456
-XX:MaxRAMPercentage=75.0
-XX:+UseG1GC
```

#### OpenJ9/IBM J9
```
-Xmx2g
-Xms256m
-Xgcpolicy:gencon
```

## Troubleshooting

### Common Issues

#### 1. "Failed to get VM flags" Error
**Cause**: jattach connection failed or jcmd not supported
**Solution**: 
- Check if process is a valid JVM
- Verify JVM supports jcmd (Java 8+)
- Check process permissions

#### 2. "Unknown" GC Type Detected
**Cause**: Unrecognized GC flags or vendor-specific GC
**Solution**:
- Check logs for vendor detection results
- Verify GC flags in command line
- May indicate unsupported JVM variant

#### 3. Empty Heap Size Values
**Cause**: Parsing failed for vendor-specific flag format
**Solution**:
- Command-line fallback should activate automatically
- Check for non-standard heap flags
- May need vendor-specific parsing enhancement

#### 4. Incorrect Vendor Detection
**Cause**: Ambiguous vendor indicators in VM output
**Solution**:
- Check system properties output
- Verify Java version and distribution
- May need manual vendor identification

### Debug Logging

Enable verbose logging to troubleshoot JVM detection:

```bash
# Enable debug logging
export KLOG_V=3

# Check logs for JVM detection
kubectl logs coroot-node-agent | grep -i jvm
```

Key log messages:
- `Detected JVM vendor: [vendor] for PID [pid]`
- `Falling back to command line parsing for PID [pid]`
- `Parsed JVM params: MaxHeap=... GC=...`

### Manual Verification

Test jcmd commands manually:

```bash
# Check if jcmd works
jcmd <pid> VM.flags

# Check system properties
jcmd <pid> VM.system_properties

# Check version info
jcmd <pid> VM.version
```

## Best Practices

### For Application Deployment

1. **Use Standard Heap Flags**: Prefer `-Xmx`/`-Xms` over vendor-specific options
2. **Explicit GC Selection**: Use `-XX:+UseG1GC` style flags
3. **JVM Version**: Use Java 8+ for best compatibility
4. **Testing**: Verify monitoring with your specific JVM distribution

### For Monitoring

1. **Log Monitoring**: Watch for JVM detection warnings
2. **Metric Validation**: Verify heap metrics are populated
3. **Vendor Awareness**: Know which JVM distributions you're running

## Future Enhancements

Planned improvements for JVM compatibility:

1. **Enhanced Size Parsing**: Better unit conversion (MB, GB)
2. **Additional Vendors**: Support for Azul, Amazon Corretto specifics
3. **JMX Fallback**: Use JMX MBeans when jcmd fails
4. **Configuration**: Allow manual vendor specification
5. **Testing Matrix**: Automated testing across JVM distributions

## Contributing

If you encounter issues with a specific JVM implementation:

1. **Report Issue**: Include JVM version, vendor, and error logs
2. **Provide Samples**: Share VM.flags and system properties output
3. **Test Cases**: Help create test cases for new JVM variants
4. **Documentation**: Update compatibility matrix

---

For questions and support, please refer to the main coroot documentation or file an issue in the GitHub repository. 