# Red Hat OpenShift Certification Guide

This document outlines the certification process and requirements for the NFS Provisioner Operator on Red Hat OpenShift.

## Certification Status

- **Current Status**: Not certified
- **Target Platform**: Red Hat OpenShift 4.19+
- **Operator Version**: 0.0.8
- **Certification Level**: Community Operator

## Prerequisites

### 1. Red Hat Partner Connect Account

To certify an operator with Red Hat, you need:

1. A Red Hat Partner Connect account
2. Access to the Red Hat Catalog certification portal
3. A valid organization/company for certification submission

Visit: https://connect.redhat.com/

### 2. Container Image Requirements

- **Registry**: quay.io (recommended) or registry.redhat.io
- **Image Scanning**: Must pass Red Hat's container vulnerability scanning
- **Image Location**: `quay.io/jooholee/nfs-provisioner-operator@sha256:d9c013967421ec72644a588a155975bf856a856f1ef38ba71dfea306fdb47acd`

### 3. Operator Bundle Requirements

The operator bundle must meet the following criteria:

#### Bundle Metadata

All required fields in `bundle/manifests/nfs-provisioner-operator.clusterserviceversion.yaml`:

- ✅ `displayName`: NFS Provisioner Operator
- ✅ `description`: Create and manage NFS Server and Provisioner
- ✅ `keywords`: nfs, storage
- ✅ `maintainers`: Jooho (ljhiyh@gmail.com)
- ✅ `links`: GitHub repository
- ✅ `provider`: Jooho Lee
- ✅ `version`: 0.0.8
- ✅ `containerImage`: Operator image reference
- ✅ `repository`: https://github.com/jooho/nfs-provisioner-operator
- ✅ `categories`: Storage
- ✅ `com.redhat.openshift.versions`: v4.19+

#### Bundle Validation

Run the operator-sdk bundle validation:

```bash
operator-sdk bundle validate ./bundle
```

Expected output:
```
All validation tests have completed successfully
```

## Certification Process

### Step 1: Prepare Bundle

1. Ensure all CSV fields are properly populated (see above)
2. Verify bundle structure follows OLM standards
3. Validate bundle using operator-sdk
4. Test on target OpenShift versions (4.19+)

### Step 2: Submit to Red Hat Partner Connect

1. Log in to Red Hat Partner Connect portal
2. Navigate to "Product Certification"
3. Select "Operator" as the product type
4. Upload operator bundle
5. Provide required documentation:
   - Operator description and use cases
   - Installation instructions
   - Support contact information
   - Testing results on OpenShift 4.19+

### Step 3: Automated Testing

Red Hat will run automated tests including:

- **Bundle Validation**: Verify OLM bundle format
- **Container Scanning**: Security vulnerability scan of operator image
- **Deployment Test**: Install operator via OLM on OpenShift 4.19+
- **Functional Test**: Verify operator creates resources correctly
- **API Compatibility**: Check Kubernetes API usage

### Step 4: Manual Review

Red Hat engineers will review:

- Code quality and best practices
- Security considerations (RBAC, SCC usage)
- Documentation completeness
- Support and maintenance plan

### Step 5: Certification Approval

Once all tests pass and manual review is complete:

1. Operator is added to Red Hat OperatorHub catalog
2. Certified badge is granted
3. Operator appears in OpenShift Console's OperatorHub

## OpenShift 4.19+ Specific Requirements

### SecurityContextConstraints (SCC)

The operator correctly handles SCC for OpenShift:

- **Detection**: Checks for SCC CRD availability
- **Creation**: Creates `nfs-provisioner` SCC when on OpenShift
- **Graceful Degradation**: Skips SCC creation on vanilla Kubernetes

Verified in: `pkg/resources/scc.go`

Test coverage: `pkg/resources/scc_test.go`, `test/integration/scc_detection_test.go`

### Platform Version Support

Declared in bundle metadata:

```yaml
annotations:
  com.redhat.openshift.versions: v4.19+
```

This annotation ensures:
- Operator is only installable on OpenShift 4.19 and newer
- Users on older versions see compatibility warnings
- OLM validates platform version before installation

## Testing Checklist

Before submitting for certification, verify:

- [ ] Operator deploys successfully on OpenShift 4.19
- [ ] All resources (Deployment, Service, PVC, StorageClass, SCC) are created
- [ ] NFSProvisioner CR reconciles successfully
- [ ] NFS server pod starts and becomes Running
- [ ] Dynamic provisioning works (create PVC using NFS StorageClass)
- [ ] SCC is applied correctly to NFS server pod
- [ ] Operator handles upgrades from previous version (0.0.7 → 0.0.8)
- [ ] Operator handles platform upgrade (4.18 → 4.19)
- [ ] Documentation is complete and accurate
- [ ] OWNERS file exists with valid maintainer information

## Post-Certification Maintenance

### Version Updates

When releasing new operator versions:

1. Update CSV with new version number
2. Update `replaces` field to previous version
3. Test upgrade path from previous certified version
4. Re-submit to Red Hat for re-certification

### OpenShift Version Support

When supporting new OpenShift versions:

1. Update `com.redhat.openshift.versions` annotation
2. Test on new OpenShift version
3. Verify no breaking API changes
4. Re-submit for certification if major changes

## Support and Resources

### Red Hat Documentation

- [Operator Certification Guide](https://redhat-connect.gitbook.io/partner-guide-for-red-hat-openshift-and-container/certify-your-operator/overview)
- [OLM Bundle Format](https://olm.operatorframework.io/docs/tasks/creating-a-bundle/)
- [OpenShift Operator Best Practices](https://docs.openshift.com/container-platform/4.19/operators/operator_sdk/osdk-about.html)

### Operator SDK

- Documentation: https://sdk.operatorframework.io/
- GitHub: https://github.com/operator-framework/operator-sdk

### Community Support

- Kubernetes Slack: #kubernetes-operators
- OpenShift Slack: #openshift-operators
- GitHub Issues: https://github.com/jooho/nfs-provisioner-operator/issues

## Troubleshooting

### Common Certification Issues

1. **Bundle Validation Fails**
   - Run `operator-sdk bundle validate ./bundle` locally
   - Fix any warnings or errors
   - Ensure all required fields are populated

2. **Container Scan Fails**
   - Review Red Hat's vulnerability report
   - Update base image to patched version
   - Rebuild and re-push operator image

3. **Deployment Test Fails**
   - Test manually on OpenShift 4.19 cluster
   - Check operator logs for errors
   - Verify RBAC permissions are sufficient

4. **SCC Issues on OpenShift**
   - Verify SCC CRD detection logic
   - Check SCC is created with correct permissions
   - Ensure service account is added to SCC users

### Getting Help

For certification questions:
- Email: ljhiyh@gmail.com
- GitHub: https://github.com/jooho/nfs-provisioner-operator
- Red Hat Partner Support (for certified partners)

## Next Steps

1. **Testing**: Deploy and validate on OpenShift 4.19+ cluster (see Phase 6 Step 4 in tasks.md)
2. **Documentation**: Complete user documentation and installation guide
3. **Partner Portal**: Create Red Hat Partner Connect account
4. **Submission**: Submit operator bundle for certification
5. **Monitoring**: Track certification progress in Partner Portal
