# integration test runner bot

- [integration test runner bot](#integration-test-runner-bot)
  - [Main features](#main-features)
    - [GitHub -\> GitLab sync](#github---gitlab-sync)
    - [GitLab PR branches](#gitlab-pr-branches)
    - [Processing GitHub events](#processing-github-events)
  - [Infrastructure](#infrastructure)
  - [Requirements](#requirements)
  - [Continuous Delivery](#continuous-delivery)
    - [Setup access to GKE](#setup-access-to-gke)
    - [Disaster Recovery](#disaster-recovery)

## Main features

### GitHub -> GitLab sync

By default all repositories from the configured GitHub organization are synced with GitLab. To select a subset of repositories to sync, set `SYNC_REPOS_LIST` env variable with a comma separated list of repositories.

### GitLab PR branches

For all repositories in the organization, a pr_XXX branch will be created in GitLab for every pull/XXX PR from GitHub.

### Processing GitHub events

Currently the following GitHub events are processed:
* `pull_request`: enabled by default, `DISABLE_PR_EVENTS_PROCESSING` disables the processing
* `push`: enabled by default, `DISABLE_PUSH_EVENTS_PROCESSING` disables the processing
* `issue_comment`: enabled by default, `DISABLE_COMMENT_EVENTS_PROCESSING` disables the processing

## Infrastructure

It's currently hosted on `company-websites` GKE Kubernetes cluster.

* A PR on `github/org/project-x` issues a Github Webhook (configured to call the website k8s cluster)
* the URL called is an API for the container `mender-test-runner` configured on the K8s cluster (currently three deployments: `test-runner-mender-io`, `repos-sync-cfengine-com`, `repos-sync-northerntechhq-com`)
* the `mender-test-runner` container get the Org from the webhook and run a sync `github/org/project-x -> gitlab/northern.tech/group/project-x`

## Requirements
1. The GH Org is mapped on [main.go](main.go)
   ```
    // Mapping https://github.com/<org> -> https://gitlab.com/Northern.tech/<group>
    var gitHubOrganizationToGitLabGroup = map[string]string{
      "mendersoftware": "Mender",
      "cfengine":       "CFEngine",
      "NorthernTechHQ": "NorthernTechHQ",
    }
   ```
1. The GH Org settings have a Webhook in place:
   1. https://github.com/organizations/NorthernTechHQ/settings/hooks
   2. Payload URL: the URL of the FQDN set on the Ingress (like `https://repos-sync.northern.tech/`)
   3. Content-type: `application/x-www-form-urlencoded`
   4. Secret: the same set on the `GITHUB_SECRET` on the [K8s secret for the pod](https://github.com/mendersoftware/sre-tools/blob/master/kubernetes/northerntechhq-repos-sync/repos-sync-northerntechhq-deployment.yaml#L46)
      which is usually stored on Mystiko along
   5. *Which events would you like to trigger this webhook?* Send me everything
2. You have the [required K8s resources](https://github.com/mendersoftware/sre-tools/tree/master/kubernetes/northerntechhq-repos-sync):
   1. Configmap for possible customizations
   2. ManagedCertificate for GCP managed Certs (for the https://repos-sync.northern.tech)
   3. The actual deployment
   4. Secrets stored on Mystiko, path `mender/saas/k8s/gke` which contains:
      1. `GITHUB_TOKEN`: the `mender-test-bot` user PAT for Github
      2. `GITHUB_SECRET`: the secret from the Webhook, like above
      3. `GITLAB_TOKEN`: the `mender-test-bot` user PAT for Gitlab
      4. `id_rsa` and `id_rsa.pub`: SSH keys for the `mender-test-bot` user
   5. Ingress configured for the new service:
      ```
        - host: repos-sync.northern.tech
          http:
            paths:
            - backend:
                service:
                  name: repos-sync-northerntechhq-com
                  port:
                    number: 8086
              pathType: ImplementationSpecific
      ```


## Continuous Delivery

Commits to the `master` branch trigger a sync with the `sre-tools` repository, committing the new Docker image's SHA256 to the file `kubernetes/mender-test-runner/test-runner-deployment.yaml`. This, in turn, triggers a new application of the Kubernetes manifest files to the cluster.

### Setup access to GKE

1. create service account with the following roles assigned: `Kubernetes Engine Developer`, `Kubernetes Engine Service Agent` and `Viewer`
2. create json key and make base64 encoded hash with removing new lines: `base64 /path/to/saved-key.json | tr -d \\n`
3. in CI/CD project settings add `GCLOUD_SERVICE_KEY` variable where value is the hash

### Disaster Recovery

Apply secret from mystico:

```bash
$ pass mender/saas/k8s/gke/secret-test-runner-mender-io.yaml | kubectl apply -f -
```

From the `sre-tools` repository:

```bash
$ kubectl apply -Rf kubernetes/mender-test-runner/
```
