# Copyright 2021 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

import os
import requests
import time

BASE_DIR = os.path.dirname(__file__)


def load_payload(filename):
    with open(os.path.join(BASE_DIR, "payloads", filename), "rb") as f:
        return f.read()


def test_pull_request_opened_from_fork(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("pull_request_opened_from_fork.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "pull_request",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Processing pull request action opened",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "git.Run: /usr/bin/git push -f -o ci.skip --set-upstream gitlab pr_140",
        "info:Created branch: workflows:pr_140",
        "gitlab.CreatePipeline: "
        + "path=Northern.tech/Mender/workflows,"
        + 'options={"ref":"pr_140","variables":'
        + '[{"key":"CI_EXTERNAL_PULL_REQUEST_IID","value":"140"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_SOURCE_REPOSITORY","value":"tranchitella/workflows"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_TARGET_REPOSITORY","value":"mendersoftware/workflows"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_NAME","value":"men-4705"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_SOURCE_BRANCH_SHA","value":"7b099b84cb50df18847027b0afa16820eab850d9"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_NAME","value":"master"},'
        + '{"key":"CI_EXTERNAL_PULL_REQUEST_TARGET_BRANCH_SHA","value":"70ab90b3932d3d008ebee56d6cfe4f3329d5ee7b"}]}',
        "debug:started pipeline for PR: ",
        "github.IsOrganizationMember: org=mendersoftware,user=tranchitella",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:workflows:140 would trigger 1 builds",
        "info:I have already commented on the pr: workflows/140, no need to keep on nagging",
    ]


def test_pull_request_opened_from_branch(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("pull_request_opened_from_branch.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "pull_request",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Processing pull request action opened",
        "debug:PR head is NOT a fork, skipping GitLab branch sync",
        "github.IsOrganizationMember: org=mendersoftware,user=lluiscampos",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "debug:syncIfOSHasEnterpriseRepo: Repository without Enterprise fork detected: (mender-docs). Not syncing",
        "info:Pull request event with action: opened",
        "info:mender-docs:1483 would trigger 0 builds",
    ]


def test_pull_request_closed(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("pull_request_closed.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "pull_request",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Processing pull request action closed",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add gitlab "
        "git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch gitlab",
        "git.Run: /usr/bin/git push gitlab --delete pr_140",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github "
        "git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git fetch github master:local",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "info:Found no changelog entries, ignoring cherry-pick suggestions",
        "github.IsOrganizationMember: org=mendersoftware,user=tranchitella",
        "debug:stopBuildsOfStalePRs: Find any running pipelines and kill mercilessly!",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:auditlogs version master is being used in master",
        "info:azure-iot-manager version master is being used in master",
        "info:create-artifact-worker version master is being used in master",
        "info:deployments version master is being used in master",
        "info:deployments-enterprise version master is being used in master",
        "info:deviceauth version master is being used in master",
        "info:deviceauth-enterprise version master is being used in master",
        "info:deviceconfig version master is being used in master",
        "info:deviceconnect version master is being used in master",
        "info:devicemonitor version master is being used in master",
        "info:gui version master is being used in master",
        "info:inventory version master is being used in master",
        "info:inventory-enterprise version master is being used in master",
        "info:mender version master is being used in master",
        "info:mender-artifact version master is being used in master",
        "info:mender-cli version master is being used in master",
        "info:mender-connect version master is being used in master",
        "info:monitor-client version master is being used in master",
        "info:mtls-ambassador version master is being used in master",
        "info:reporting version master is being used in master",
        "info:tenantadm version master is being used in master",
        "info:useradm version master is being used in master",
        "info:useradm-enterprise version master is being used in master",
        "info:workflows-enterprise version master is being used in master",
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"pending","username":"mender-test-bot"}',
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"running","username":"mender-test-bot"}',
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "info:syncIfOSHasEnterpriseRepo: Merge to (master) in an OS repository "
        "detected. Syncing the repositories...",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add opensource "
        "git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add enterprise "
        "git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add mender-test-bot "
        "git@github.com:/mender-test-bot/workflows-enterprise.git",
        "git.Run: /usr/bin/git config --add user.name mender-test-bot",
        "git.Run: /usr/bin/git config --add user.email mender@northern.tech",
        "git.Run: /usr/bin/git fetch opensource master",
        "git.Run: /usr/bin/git fetch enterprise master:mergeostoent_140",
        "git.Run: /usr/bin/git checkout mergeostoent_140",
        "debug:Trying to Merge OS base branch: (master) including PR: (140) into "
        "Enterprise: (master)",
        "git.Run: /usr/bin/git merge -m Merge OS base branch: (master) including PR: "
        "(140) into Enterprise: (master) opensource/master",
        "git.Run: /usr/bin/git push --set-upstream mender-test-bot mergeostoent_140",
        "info:Merged branch: opensource/workflows/master into "
        "enterprise/workflows/master in the Enterprise repo",
        "github.CreatePullRequest: "
        'org=mendersoftware,repo=workflows-enterprise,pr={"title":"[Bot] Improve '
        'logging","head":"mender-test-bot:mergeostoent_140","base":"master","body":"Original '
        "PR: https://github.com/mendersoftware/workflows/pull/140\\n\\nChangelog: "
        "none\\r\\n\\r\\nSigned-off-by: Fabio Tranchitella "
        '\\u003cfabio.tranchitella@northern.tech\\u003e","maintainer_can_modify":true}',
        "info:syncIfOSHasEnterpriseRepo: Created PR: 0 on Enterprise/workflows/master",
        "debug:syncIfOSHasEnterpriseRepo: Created PR: "
        "id=666510619,number=140,title=Improve logging",
        "debug:Trying to @mention the user in the newly created PR",
        "debug:userName: tranchitella",
        "github.CreateComment: "
        'org=mendersoftware,repo=workflows-enterprise,number=0,comment={"body":"@tranchitella '
        'I have created a PR for you, ready to merge as soon as tests are passed"}',
        "info:Pull request event with action: closed",
        "info:workflows:140 would trigger 0 builds",
    ]


def test_push_mendersoftware(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("push.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "push",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Got push event :: repo workflows-enterprise :: ref refs/heads/master",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows-enterprise",
        "git.Run: /usr/bin/git fetch github",
        "git.Run: /usr/bin/git checkout -b master github/master",
        "git.Run: /usr/bin/git push -f gitlab master",
        "info:Pushed ref to GitLab: workflows-enterprise:refs/heads/master",
    ]


def test_push_cfengine(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("push_cfengine.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "push",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Got push event :: repo website :: ref refs/heads/master",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/cfengine/website.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/CFEngine/website",
        "git.Run: /usr/bin/git fetch github",
        "git.Run: /usr/bin/git checkout -b master github/master",
        "git.Run: /usr/bin/git push -f gitlab master",
        "info:Pushed ref to GitLab: website:refs/heads/master",
    ]


def test_push_mender_qa_repo(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("push_mender_qa_repo.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "push",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "debug:Got push event :: repo mender-qa :: ref refs/heads/master",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/mender-qa.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/mender-qa",
        "git.Run: /usr/bin/git fetch github",
        "git.Run: /usr/bin/git checkout -b master github/master",
        "git.Run: /usr/bin/git push -f -o ci.skip gitlab master",
        "info:Pushed ref to GitLab: mender-qa:refs/heads/master",
    ]


def test_issue_comment(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("issue_comment.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "issue_comment",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "github.IsOrganizationMember: org=mendersoftware,user=alfrunes",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:deviceconnect/master is being used in the following integration: "
        "[master]",
        "info:the following integration branches: [master] are using "
        "deviceconnect/master",
        "info:deviceconnect:109 will trigger 1 builds",
        "info:1: (main.buildOptions) {\n"
        ' pr: (string) (len=3) "109",\n'
        ' repo: (string) (len=13) "deviceconnect",\n'
        ' baseBranch: (string) (len=6) "master",\n'
        ' commitSHA: (string) (len=40) "ddc66080a35f0d1d4bc1d3ef589a8226b2c9a02b",\n'
        " makeQEMU: (bool) false\n"
        "}\n"
        "\n",
        "info:auditlogs version master is being used in master",
        "info:azure-iot-manager version master is being used in master",
        "info:create-artifact-worker version master is being used in master",
        "info:deployments version master is being used in master",
        "info:deployments-enterprise version master is being used in master",
        "info:deviceauth version master is being used in master",
        "info:deviceauth-enterprise version master is being used in master",
        "info:deviceconfig version master is being used in master",
        "info:devicemonitor version master is being used in master",
        "info:gui version master is being used in master",
        "info:inventory version master is being used in master",
        "info:inventory-enterprise version master is being used in master",
        "info:mender version master is being used in master",
        "info:mender-artifact version master is being used in master",
        "info:mender-cli version master is being used in master",
        "info:mender-connect version master is being used in master",
        "info:monitor-client version master is being used in master",
        "info:mtls-ambassador version master is being used in master",
        "info:reporting version master is being used in master",
        "info:tenantadm version master is being used in master",
        "info:useradm version master is being used in master",
        "info:useradm-enterprise version master is being used in master",
        "info:workflows version master is being used in master",
        "info:workflows-enterprise version master is being used in master",
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"pending","username":"mender-test-bot"}',
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"running","username":"mender-test-bot"}',
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "info:Creating pipeline in project Northern.tech/Mender/mender-qa:master with "
        "variables: AUDITLOGS_REV:master, AZURE_IOT_MANAGER_REV:master, "
        "BUILD_BEAGLEBONEBLACK:, BUILD_CLIENT:false, "
        "BUILD_QEMUX86_64_BIOS_GRUB:, BUILD_QEMUX86_64_BIOS_GRUB_GPT:, "
        "BUILD_QEMUX86_64_UEFI_GRUB:, BUILD_VEXPRESS_QEMU:, "
        "BUILD_VEXPRESS_QEMU_FLASH:, BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, "
        "CREATE_ARTIFACT_WORKER_REV:master, DEPLOYMENTS_ENTERPRISE_REV:master, "
        "DEPLOYMENTS_REV:master, DEVICEAUTH_ENTERPRISE_REV:master, DEVICEAUTH_REV:master, "
        "DEVICECONFIG_REV:master, "
        "DEVICECONNECT_REV:pull/109/head, DEVICEMONITOR_REV:master, GUI_REV:master, "
        "INTEGRATION_REV:master, INVENTORY_ENTERPRISE_REV:master, "
        "INVENTORY_REV:master, MENDER_ARTIFACT_REV:master, MENDER_CLI_REV:master, "
        "MENDER_CONNECT_REV:master, MENDER_REV:master, MONITOR_CLIENT_REV:master, "
        "MTLS_AMBASSADOR_REV:master, REPORTING_REV:master, "
        "RUN_INTEGRATION_TESTS:true, TENANTADM_REV:master, "
        "TEST_QEMUX86_64_BIOS_GRUB:, TEST_QEMUX86_64_BIOS_GRUB_GPT:, "
        "TEST_QEMUX86_64_UEFI_GRUB:, TEST_VEXPRESS_QEMU:, TEST_VEXPRESS_QEMU_FLASH:, "
        "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, USERADM_ENTERPRISE_REV:master, "
        "USERADM_REV:master, WORKFLOWS_ENTERPRISE_REV:master, WORKFLOWS_REV:master, ",
        "gitlab.CreatePipeline: "
        + "path=Northern.tech/Mender/mender-qa,"
        + 'options={"ref":"master","variables":'
        + '[{"key":"AUDITLOGS_REV","value":"master"},'
        + '{"key":"AZURE_IOT_MANAGER_REV","value":"master"},'
        + '{"key":"BUILD_BEAGLEBONEBLACK","value":""},'
        + '{"key":"BUILD_CLIENT","value":"false"},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB","value":""},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB_GPT","value":""},'
        + '{"key":"BUILD_QEMUX86_64_UEFI_GRUB","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU_FLASH","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},'
        + '{"key":"CREATE_ARTIFACT_WORKER_REV","value":"master"},'
        + '{"key":"DEPLOYMENTS_ENTERPRISE_REV","value":"master"},'
        + '{"key":"DEPLOYMENTS_REV","value":"master"},'
        + '{"key":"DEVICEAUTH_ENTERPRISE_REV","value":"master"},'
        + '{"key":"DEVICEAUTH_REV","value":"master"},'
        + '{"key":"DEVICECONFIG_REV","value":"master"},'
        + '{"key":"DEVICECONNECT_REV","value":"pull/109/head"},'
        + '{"key":"DEVICEMONITOR_REV","value":"master"},'
        + '{"key":"GUI_REV","value":"master"},'
        + '{"key":"INTEGRATION_REV","value":"master"},'
        + '{"key":"INVENTORY_ENTERPRISE_REV","value":"master"},'
        + '{"key":"INVENTORY_REV","value":"master"},'
        + '{"key":"MENDER_ARTIFACT_REV","value":"master"},'
        + '{"key":"MENDER_CLI_REV","value":"master"},'
        + '{"key":"MENDER_CONNECT_REV","value":"master"},'
        + '{"key":"MENDER_REV","value":"master"},'
        + '{"key":"MONITOR_CLIENT_REV","value":"master"},'
        + '{"key":"MTLS_AMBASSADOR_REV","value":"master"},'
        + '{"key":"REPORTING_REV","value":"master"},'
        + '{"key":"RUN_INTEGRATION_TESTS","value":"true"},'
        + '{"key":"TENANTADM_REV","value":"master"},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB","value":""},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB_GPT","value":""},'
        + '{"key":"TEST_QEMUX86_64_UEFI_GRUB","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU_FLASH","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},'
        + '{"key":"USERADM_ENTERPRISE_REV","value":"master"},'
        + '{"key":"USERADM_REV","value":"master"},'
        + '{"key":"WORKFLOWS_ENTERPRISE_REV","value":"master"},'
        + '{"key":"WORKFLOWS_REV","value":"master"}]}',
        "info:Created pipeline: ",
        "github.CreateComment: "
        'org=mendersoftware,repo=deviceconnect,number=109,comment={"body":"\\nHello '
        ":smile_cat: I created a pipeline for you here: "
        "[Pipeline-0]()\\n\\n\\u003cdetails\\u003e\\n    \\u003csummary\\u003eBuild "
        "Configuration Matrix\\u003c/summary\\u003e\\u003cp\\u003e\\n\\n| Key   | "
        "Value |\\n| ----- | ----- |\\n| AUDITLOGS_REV | master |\\n| "
        "AZURE_IOT_MANAGER_REV | master |\\n| "
        "BUILD_CLIENT | "
        "false |\\n| CREATE_ARTIFACT_WORKER_REV | master |\\n| "
        "DEPLOYMENTS_ENTERPRISE_REV | master |\\n| DEPLOYMENTS_REV | master |\\n| "
        "DEVICEAUTH_ENTERPRISE_REV | master |\\n| DEVICEAUTH_REV | master |\\n| "
        "DEVICECONFIG_REV | master |\\n| "
        "DEVICECONNECT_REV | pull/109/head |\\n| DEVICEMONITOR_REV | master |\\n| "
        "GUI_REV | master |\\n| INTEGRATION_REV | master |\\n| "
        "INVENTORY_ENTERPRISE_REV | master |\\n| INVENTORY_REV | master |\\n| "
        "MENDER_ARTIFACT_REV | master |\\n| MENDER_CLI_REV | master |\\n| "
        "MENDER_CONNECT_REV | master |\\n| MENDER_REV | master |\\n| "
        "MONITOR_CLIENT_REV | master |\\n| MTLS_AMBASSADOR_REV | master |\\n| "
        "REPORTING_REV | master |\\n| RUN_INTEGRATION_TESTS | true |\\n| "
        "TENANTADM_REV | master |\\n| USERADM_ENTERPRISE_REV | master |\\n| "
        "USERADM_REV | master |\\n| WORKFLOWS_ENTERPRISE_REV | master |\\n| "
        "WORKFLOWS_REV | master |\\n\\n\\n "
        '\\u003c/p\\u003e\\u003c/details\\u003e\\n"}',
    ]


def test_issue_comment___pr(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("issue_comment___pr.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "issue_comment",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "github.IsOrganizationMember: org=mendersoftware,user=alfrunes",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:deviceconnect/master is being used in the following integration: "
        "[master]",
        "info:the following integration branches: [master] are using "
        "deviceconnect/master",
        "info:deviceconnect:109 will trigger 1 builds",
        "info:1: (main.buildOptions) {\n"
        ' pr: (string) (len=3) "109",\n'
        ' repo: (string) (len=13) "deviceconnect",\n'
        ' baseBranch: (string) (len=6) "master",\n'
        ' commitSHA: (string) (len=40) "ddc66080a35f0d1d4bc1d3ef589a8226b2c9a02b",\n'
        " makeQEMU: (bool) false\n"
        "}\n"
        "\n",
        "info:auditlogs version master is being used in master",
        "info:azure-iot-manager version master is being used in master",
        "info:create-artifact-worker version master is being used in master",
        "info:deployments version master is being used in master",
        "info:deployments-enterprise version master is being used in master",
        "info:deviceauth version master is being used in master",
        "info:deviceauth-enterprise version master is being used in master",
        "info:deviceconfig version master is being used in master",
        "info:gui version master is being used in master",
        "info:inventory version master is being used in master",
        "info:inventory-enterprise version master is being used in master",
        "info:mender-artifact version master is being used in master",
        "info:mender-cli version master is being used in master",
        "info:monitor-client version master is being used in master",
        "info:mtls-ambassador version master is being used in master",
        "info:reporting version master is being used in master",
        "info:tenantadm version master is being used in master",
        "info:useradm version master is being used in master",
        "info:useradm-enterprise version master is being used in master",
        "info:workflows version master is being used in master",
        "info:workflows-enterprise version master is being used in master",
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"pending","username":"mender-test-bot"}',
        "gitlab.ListProjectPipelines: "
        'path=Northern.tech/Mender/mender-qa,options={"status":"running","username":"mender-test-bot"}',
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "info:Creating pipeline in project Northern.tech/Mender/mender-qa:master with "
        "variables: AUDITLOGS_REV:master, AZURE_IOT_MANAGER_REV:master, "
        "BUILD_BEAGLEBONEBLACK:, BUILD_CLIENT:false, "
        "BUILD_QEMUX86_64_BIOS_GRUB:, BUILD_QEMUX86_64_BIOS_GRUB_GPT:, "
        "BUILD_QEMUX86_64_UEFI_GRUB:, BUILD_VEXPRESS_QEMU:, "
        "BUILD_VEXPRESS_QEMU_FLASH:, BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, "
        "CREATE_ARTIFACT_WORKER_REV:master, DEPLOYMENTS_ENTERPRISE_REV:master, "
        "DEPLOYMENTS_REV:master, DEVICEAUTH_ENTERPRISE_REV:master, DEVICEAUTH_REV:master, "
        "DEVICECONFIG_REV:master, "
        "DEVICECONNECT_REV:pull/109/head, DEVICEMONITOR_REV:pull/12/head, GUI_REV:master, "
        "INTEGRATION_REV:master, INVENTORY_ENTERPRISE_REV:master, "
        "INVENTORY_REV:master, MENDER_ARTIFACT_REV:master, MENDER_CLI_REV:master, "
        "MENDER_CONNECT_REV:pull/4/head, MENDER_REV:3.1.x, MONITOR_CLIENT_REV:master, "
        "MTLS_AMBASSADOR_REV:master, REPORTING_REV:master, "
        "RUN_INTEGRATION_TESTS:true, TENANTADM_REV:master, "
        "TEST_QEMUX86_64_BIOS_GRUB:, TEST_QEMUX86_64_BIOS_GRUB_GPT:, "
        "TEST_QEMUX86_64_UEFI_GRUB:, TEST_VEXPRESS_QEMU:, TEST_VEXPRESS_QEMU_FLASH:, "
        "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, USERADM_ENTERPRISE_REV:master, "
        "USERADM_REV:master, WORKFLOWS_ENTERPRISE_REV:master, WORKFLOWS_REV:master, ",
        "gitlab.CreatePipeline: "
        + "path=Northern.tech/Mender/mender-qa,"
        + 'options={"ref":"master","variables":'
        + '[{"key":"AUDITLOGS_REV","value":"master"},'
        + '{"key":"AZURE_IOT_MANAGER_REV","value":"master"},'
        + '{"key":"BUILD_BEAGLEBONEBLACK","value":""},'
        + '{"key":"BUILD_CLIENT","value":"false"},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB","value":""},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB_GPT","value":""},'
        + '{"key":"BUILD_QEMUX86_64_UEFI_GRUB","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU_FLASH","value":""},'
        + '{"key":"BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},'
        + '{"key":"CREATE_ARTIFACT_WORKER_REV","value":"master"},'
        + '{"key":"DEPLOYMENTS_ENTERPRISE_REV","value":"master"},'
        + '{"key":"DEPLOYMENTS_REV","value":"master"},'
        + '{"key":"DEVICEAUTH_ENTERPRISE_REV","value":"master"},'
        + '{"key":"DEVICEAUTH_REV","value":"master"},'
        + '{"key":"DEVICECONFIG_REV","value":"master"},'
        + '{"key":"DEVICECONNECT_REV","value":"pull/109/head"},'
        + '{"key":"DEVICEMONITOR_REV","value":"pull/12/head"},'
        + '{"key":"GUI_REV","value":"master"},'
        + '{"key":"INTEGRATION_REV","value":"master"},'
        + '{"key":"INVENTORY_ENTERPRISE_REV","value":"master"},'
        + '{"key":"INVENTORY_REV","value":"master"},'
        + '{"key":"MENDER_ARTIFACT_REV","value":"master"},'
        + '{"key":"MENDER_CLI_REV","value":"master"},'
        + '{"key":"MENDER_CONNECT_REV","value":"pull/4/head"},'
        + '{"key":"MENDER_REV","value":"3.1.x"},'
        + '{"key":"MONITOR_CLIENT_REV","value":"master"},'
        + '{"key":"MTLS_AMBASSADOR_REV","value":"master"},'
        + '{"key":"REPORTING_REV","value":"master"},'
        + '{"key":"RUN_INTEGRATION_TESTS","value":"true"},'
        + '{"key":"TENANTADM_REV","value":"master"},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB","value":""},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB_GPT","value":""},'
        + '{"key":"TEST_QEMUX86_64_UEFI_GRUB","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU_FLASH","value":""},'
        + '{"key":"TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},'
        + '{"key":"USERADM_ENTERPRISE_REV","value":"master"},'
        + '{"key":"USERADM_REV","value":"master"},'
        + '{"key":"WORKFLOWS_ENTERPRISE_REV","value":"master"},'
        + '{"key":"WORKFLOWS_REV","value":"master"}]}',
        "info:Created pipeline: ",
        "github.CreateComment: "
        'org=mendersoftware,repo=deviceconnect,number=109,comment={"body":"\\nHello '
        ":smile_cat: I created a pipeline for you here: "
        "[Pipeline-0]()\\n\\n\\u003cdetails\\u003e\\n    \\u003csummary\\u003eBuild "
        "Configuration Matrix\\u003c/summary\\u003e\\u003cp\\u003e\\n\\n| Key   | "
        "Value |\\n| ----- | ----- |\\n| AUDITLOGS_REV | master |\\n| "
        "AZURE_IOT_MANAGER_REV | master |\\n| "
        "BUILD_CLIENT | "
        "false |\\n| CREATE_ARTIFACT_WORKER_REV | master |\\n| "
        "DEPLOYMENTS_ENTERPRISE_REV | master |\\n| DEPLOYMENTS_REV | master |\\n| "
        "DEVICEAUTH_ENTERPRISE_REV | master |\\n| DEVICEAUTH_REV | master |\\n| "
        "DEVICECONFIG_REV | master |\\n| "
        "DEVICECONNECT_REV | pull/109/head |\\n| DEVICEMONITOR_REV | pull/12/head |\\n| "
        "GUI_REV | master |\\n| INTEGRATION_REV | master |\\n| "
        "INVENTORY_ENTERPRISE_REV | master |\\n| INVENTORY_REV | master |\\n| "
        "MENDER_ARTIFACT_REV | master |\\n| MENDER_CLI_REV | master |\\n| "
        "MENDER_CONNECT_REV | pull/4/head |\\n| MENDER_REV | 3.1.x |\\n| "
        "MONITOR_CLIENT_REV | master |\\n| MTLS_AMBASSADOR_REV | master |\\n| "
        "REPORTING_REV | master |\\n| RUN_INTEGRATION_TESTS | true |\\n| "
        "TENANTADM_REV | master |\\n| USERADM_ENTERPRISE_REV | master |\\n| "
        "USERADM_REV | master |\\n| WORKFLOWS_ENTERPRISE_REV | master |\\n| "
        "WORKFLOWS_REV | master |\\n\\n\\n "
        '\\u003c/p\\u003e\\u003c/details\\u003e\\n"}',
    ]


def test_issue_comment_minor_series(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("issue_comment_minor_series.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "issue_comment",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "github.IsOrganizationMember: org=mendersoftware,user=kacf",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:mender/3.1.x is being used in the following integration: [3.1.x]",
        "info:the following integration branches: [3.1.x] are using mender/3.1.x",
        "info:mender:865 will trigger 1 builds",
        'info:1: (main.buildOptions) {\n pr: (string) (len=3) "865",\n repo: (string) (len=6) "mender",\n baseBranch: (string) (len=5) "3.1.x",\n commitSHA: (string) (len=40) "75ad5f739a6e0bd3367e92d846521a85a4e8bb35",\n makeQEMU: (bool) true\n}\n\n',
        "info:auditlogs version 2.0.x is being used in 3.1.x",
        "info:azure-iot-manager version  is being used in 3.1.x",
        "info:create-artifact-worker version 1.0.x is being used in 3.1.x",
        "info:deployments version 4.0.x is being used in 3.1.x",
        "info:deployments-enterprise version 4.0.x is being used in 3.1.x",
        "info:deviceauth version 3.1.x is being used in 3.1.x",
        "info:deviceauth-enterprise version  is being used in 3.1.x",
        "info:deviceconfig version 1.1.x is being used in 3.1.x",
        "info:deviceconnect version 1.2.x is being used in 3.1.x",
        "info:devicemonitor version 1.0.x is being used in 3.1.x",
        "info:gui version 3.1.x is being used in 3.1.x",
        "info:inventory version 4.0.x is being used in 3.1.x",
        "info:inventory-enterprise version 4.0.x is being used in 3.1.x",
        "info:mender-artifact version 3.6.x is being used in 3.1.x",
        "info:mender-cli version 1.7.x is being used in 3.1.x",
        "info:mender-connect version 1.2.x is being used in 3.1.x",
        "info:monitor-client version 1.0.x is being used in 3.1.x",
        "info:mtls-ambassador version 1.0.x is being used in 3.1.x",
        "info:reporting version master is being used in 3.1.x",
        "info:tenantadm version 3.3.x is being used in 3.1.x",
        "info:useradm version 1.16.x is being used in 3.1.x",
        "info:useradm-enterprise version 1.16.x is being used in 3.1.x",
        "info:workflows version 2.1.x is being used in 3.1.x",
        "info:workflows-enterprise version 2.1.x is being used in 3.1.x",
        'gitlab.ListProjectPipelines: path=Northern.tech/Mender/mender-qa,options={"status":"pending","username":"mender-test-bot"}',
        'gitlab.ListProjectPipelines: path=Northern.tech/Mender/mender-qa,options={"status":"running","username":"mender-test-bot"}',
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "gitlab.GetPipelineVariables: path=Northern.tech/Mender/mender-qa,id=1",
        "info:Creating pipeline in project Northern.tech/Mender/mender-qa:master with variables: "
        "AUDITLOGS_REV:2.0.x, "
        "AZURE_IOT_MANAGER_REV:, "
        "BUILD_BEAGLEBONEBLACK:true, "
        "BUILD_CLIENT:true, "
        "BUILD_QEMUX86_64_BIOS_GRUB:true, "
        "BUILD_QEMUX86_64_BIOS_GRUB_GPT:true, "
        "BUILD_QEMUX86_64_UEFI_GRUB:true, "
        "BUILD_VEXPRESS_QEMU:true, "
        "BUILD_VEXPRESS_QEMU_FLASH:true, "
        "BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:true, "
        "CREATE_ARTIFACT_WORKER_REV:1.0.x, "
        "DEPLOYMENTS_ENTERPRISE_REV:4.0.x, "
        "DEPLOYMENTS_REV:4.0.x, "
        "DEVICEAUTH_ENTERPRISE_REV:, "
        "DEVICEAUTH_REV:3.1.x, "
        "DEVICECONFIG_REV:1.1.x, "
        "DEVICECONNECT_REV:1.2.x, "
        "DEVICEMONITOR_REV:1.0.x, "
        "GUI_REV:3.1.x, "
        "INTEGRATION_REV:3.1.x, "
        "INVENTORY_ENTERPRISE_REV:4.0.x, "
        "INVENTORY_REV:4.0.x, "
        "MENDER_ARTIFACT_REV:3.6.x, "
        "MENDER_CLI_REV:1.7.x, "
        "MENDER_CONNECT_REV:1.2.x, "
        "MENDER_REV:pull/865/head, "
        "META_MENDER_REV:dunfell, "
        "META_OPENEMBEDDED_REV:dunfell, "
        "META_RASPBERRYPI_REV:dunfell, "
        "MONITOR_CLIENT_REV:1.0.x, "
        "MTLS_AMBASSADOR_REV:1.0.x, "
        "POKY_REV:dunfell, "
        "REPORTING_REV:master, "
        "RUN_INTEGRATION_TESTS:true, "
        "TENANTADM_REV:3.3.x, "
        "TEST_QEMUX86_64_BIOS_GRUB:true, "
        "TEST_QEMUX86_64_BIOS_GRUB_GPT:true, "
        "TEST_QEMUX86_64_UEFI_GRUB:true, "
        "TEST_VEXPRESS_QEMU:true, "
        "TEST_VEXPRESS_QEMU_FLASH:true, "
        "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:true, "
        "USERADM_ENTERPRISE_REV:1.16.x, "
        "USERADM_REV:1.16.x, "
        "WORKFLOWS_ENTERPRISE_REV:2.1.x, "
        "WORKFLOWS_REV:2.1.x, ",
        'gitlab.CreatePipeline: '
+ 'path=Northern.tech/Mender/mender-qa,'
+'options={"ref":"master","variables":'
+'[{"key":"AUDITLOGS_REV","value":"2.0.x"},'
        + '{"key":"AZURE_IOT_MANAGER_REV","value":""},'
        + '{"key":"BUILD_BEAGLEBONEBLACK","value":"true"},'
        + '{"key":"BUILD_CLIENT","value":"true"},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB","value":"true"},'
        + '{"key":"BUILD_QEMUX86_64_BIOS_GRUB_GPT","value":"true"},'
        + '{"key":"BUILD_QEMUX86_64_UEFI_GRUB","value":"true"},'
        + '{"key":"BUILD_VEXPRESS_QEMU","value":"true"},'
        + '{"key":"BUILD_VEXPRESS_QEMU_FLASH","value":"true"},'
        + '{"key":"BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":"true"},'
        + '{"key":"CREATE_ARTIFACT_WORKER_REV","value":"1.0.x"},'
        + '{"key":"DEPLOYMENTS_ENTERPRISE_REV","value":"4.0.x"},'
        + '{"key":"DEPLOYMENTS_REV","value":"4.0.x"},'
        + '{"key":"DEVICEAUTH_ENTERPRISE_REV","value":""},'
        + '{"key":"DEVICEAUTH_REV","value":"3.1.x"},'
        + '{"key":"DEVICECONFIG_REV","value":"1.1.x"},'
        + '{"key":"DEVICECONNECT_REV","value":"1.2.x"},'
        + '{"key":"DEVICEMONITOR_REV","value":"1.0.x"},'
        + '{"key":"GUI_REV","value":"3.1.x"},'
        + '{"key":"INTEGRATION_REV","value":"3.1.x"},'
        + '{"key":"INVENTORY_ENTERPRISE_REV","value":"4.0.x"},'
        + '{"key":"INVENTORY_REV","value":"4.0.x"},'
        + '{"key":"MENDER_ARTIFACT_REV","value":"3.6.x"},'
        + '{"key":"MENDER_CLI_REV","value":"1.7.x"},'
        + '{"key":"MENDER_CONNECT_REV","value":"1.2.x"},'
        + '{"key":"MENDER_REV","value":"pull/865/head"},'
        + '{"key":"META_MENDER_REV","value":"dunfell"},'
        + '{"key":"META_OPENEMBEDDED_REV","value":"dunfell"},'
        + '{"key":"META_RASPBERRYPI_REV","value":"dunfell"},'
        + '{"key":"MONITOR_CLIENT_REV","value":"1.0.x"},'
        + '{"key":"MTLS_AMBASSADOR_REV","value":"1.0.x"},'
        + '{"key":"POKY_REV","value":"dunfell"},'
        + '{"key":"REPORTING_REV","value":"master"},'
        + '{"key":"RUN_INTEGRATION_TESTS","value":"true"},'
        + '{"key":"TENANTADM_REV","value":"3.3.x"},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB","value":"true"},'
        + '{"key":"TEST_QEMUX86_64_BIOS_GRUB_GPT","value":"true"},'
        + '{"key":"TEST_QEMUX86_64_UEFI_GRUB","value":"true"},'
        + '{"key":"TEST_VEXPRESS_QEMU","value":"true"},'
        + '{"key":"TEST_VEXPRESS_QEMU_FLASH","value":"true"},'
        + '{"key":"TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":"true"},'
        + '{"key":"USERADM_ENTERPRISE_REV","value":"1.16.x"},'
        + '{"key":"USERADM_REV","value":"1.16.x"},'
        + '{"key":"WORKFLOWS_ENTERPRISE_REV","value":"2.1.x"},'
        + '{"key":"WORKFLOWS_REV","value":"2.1.x"}]}',
        "info:Created pipeline: ",
        'github.CreateComment: org=mendersoftware,repo=mender,number=865,comment='
        '{"body":"\\nHello :smile_cat: I created a pipeline for you here: '
        '[Pipeline-0]()\\n\\n\\u003cdetails\\u003e\\n    \\u003csummary\\u003eBuild Configuration Matrix\\u003c/summary\\u003e\\u003cp\\u003e\\n\\n| '
        'Key   | Value |\\n| '
        '----- | ----- |\\n| '
        'AUDITLOGS_REV | 2.0.x |\\n| '
        'BUILD_BEAGLEBONEBLACK | true |\\n| '
        'BUILD_CLIENT | true |\\n| '
        'BUILD_QEMUX86_64_BIOS_GRUB | true |\\n| '
        'BUILD_QEMUX86_64_BIOS_GRUB_GPT | true |\\n| '
        'BUILD_QEMUX86_64_UEFI_GRUB | true |\\n| '
        'BUILD_VEXPRESS_QEMU | true |\\n| '
        'BUILD_VEXPRESS_QEMU_FLASH | true |\\n| '
        'BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB | true |\\n| '
        'CREATE_ARTIFACT_WORKER_REV | 1.0.x |\\n| '
        'DEPLOYMENTS_ENTERPRISE_REV | 4.0.x |\\n| '
        'DEPLOYMENTS_REV | 4.0.x |\\n| '
        'DEVICEAUTH_REV | 3.1.x |\\n| '
        'DEVICECONFIG_REV | 1.1.x |\\n| '
        'DEVICECONNECT_REV | 1.2.x |\\n| '
        'DEVICEMONITOR_REV | 1.0.x |\\n| '
        'GUI_REV | 3.1.x |\\n| '
        'INTEGRATION_REV | 3.1.x |\\n| '
        'INVENTORY_ENTERPRISE_REV | 4.0.x |\\n| '
        'INVENTORY_REV | 4.0.x |\\n| '
        'MENDER_ARTIFACT_REV | 3.6.x |\\n| '
        'MENDER_CLI_REV | 1.7.x |\\n| '
        'MENDER_CONNECT_REV | 1.2.x |\\n| '
        'MENDER_REV | pull/865/head |\\n| '
        'META_MENDER_REV | dunfell |\\n| '
        'META_OPENEMBEDDED_REV | dunfell |\\n| '
        'META_RASPBERRYPI_REV | dunfell |\\n| '
        'MONITOR_CLIENT_REV | 1.0.x |\\n| '
        'MTLS_AMBASSADOR_REV | 1.0.x |\\n| '
        'POKY_REV | dunfell |\\n| '
        'REPORTING_REV | master |\\n| '
        'RUN_INTEGRATION_TESTS | true |\\n| '
        'TENANTADM_REV | 3.3.x |\\n| '
        'TEST_QEMUX86_64_BIOS_GRUB | true |\\n| '
        'TEST_QEMUX86_64_BIOS_GRUB_GPT | true |\\n| '
        'TEST_QEMUX86_64_UEFI_GRUB | true |\\n| '
        'TEST_VEXPRESS_QEMU | true |\\n| '
        'TEST_VEXPRESS_QEMU_FLASH | true |\\n| '
        'TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB | true |\\n| '
        'USERADM_ENTERPRISE_REV | 1.16.x |\\n| '
        'USERADM_REV | 1.16.x |\\n| '
        'WORKFLOWS_ENTERPRISE_REV | 2.1.x |\\n| '
        'WORKFLOWS_REV | 2.1.x |\\n\\n\\n '
        '\\u003c/p\\u003e\\u003c/details\\u003e\\n"}',
    ]


def test_cherrypick(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("pr_cherry_pick_comment.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "issue_comment",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 202
    #
    res = requests.get(integration_test_runner_url + "/logs")
    assert res.status_code == 200
    assert res.json() == [
        "github.IsOrganizationMember: org=mendersoftware,user=oleorhagen",
        "info:Attempting to cherry-pick the changes in PR: mender/864",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add mendersoftware "
        "git@github.com:/mendersoftware/mender.git",
        "git.Run: /usr/bin/git fetch mendersoftware",
        "git.Run: /usr/bin/git checkout mendersoftware/3.1.x",
        "git.Run: /usr/bin/git checkout -b cherry-3.1.x-logbuffering",
        "git.Run: /usr/bin/git cherry-pick -x "
        "f48250b19fae7ba72de2439c20a0fc678afa9a87 "
        "^4c6d93ba936031ee00d9c115ef2dc61597bc1296",
        "git.Run: /usr/bin/git push mendersoftware "
        "cherry-3.1.x-logbuffering:cherry-3.1.x-logbuffering",
        "github.CreatePullRequest: "
        'org=mendersoftware,repo=mender,pr={"title":"[Cherry 3.1.x]: MEN-5098: '
        "Capture and pretty print output from scripts "
        'executed","head":"cherry-3.1.x-logbuffering","base":"3.1.x","body":"Cherry '
        'pick of PR: #864\\nFor you  :)","maintainer_can_modify":true}',
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add mendersoftware "
        "git@github.com:/mendersoftware/mender.git",
        "git.Run: /usr/bin/git fetch mendersoftware",
        "git.Run: /usr/bin/git checkout mendersoftware/3.0.x",
        "git.Run: /usr/bin/git checkout -b cherry-3.0.x-logbuffering",
        "git.Run: /usr/bin/git cherry-pick -x "
        "f48250b19fae7ba72de2439c20a0fc678afa9a87 "
        "^4c6d93ba936031ee00d9c115ef2dc61597bc1296",
        "git.Run: /usr/bin/git push mendersoftware "
        "cherry-3.0.x-logbuffering:cherry-3.0.x-logbuffering",
        "github.CreatePullRequest: "
        'org=mendersoftware,repo=mender,pr={"title":"[Cherry 3.0.x]: MEN-5098: '
        "Capture and pretty print output from scripts "
        'executed","head":"cherry-3.0.x-logbuffering","base":"3.0.x","body":"Cherry '
        'pick of PR: #864\\nFor you  :)","maintainer_can_modify":true}',
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add mendersoftware "
        "git@github.com:/mendersoftware/mender.git",
        "git.Run: /usr/bin/git fetch mendersoftware",
        "git.Run: /usr/bin/git checkout mendersoftware/2.6.x",
        "git.Run: /usr/bin/git checkout -b cherry-2.6.x-logbuffering",
        "git.Run: /usr/bin/git cherry-pick -x "
        "f48250b19fae7ba72de2439c20a0fc678afa9a87 "
        "^4c6d93ba936031ee00d9c115ef2dc61597bc1296",
        "git.Run: /usr/bin/git push mendersoftware "
        "cherry-2.6.x-logbuffering:cherry-2.6.x-logbuffering",
        "github.CreatePullRequest: "
        'org=mendersoftware,repo=mender,pr={"title":"[Cherry 2.6.x]: MEN-5098: '
        "Capture and pretty print output from scripts "
        'executed","head":"cherry-2.6.x-logbuffering","base":"2.6.x","body":"Cherry '
        'pick of PR: #864\\nFor you  :)","maintainer_can_modify":true}',
        "github.CreateComment: "
        'org=mendersoftware,repo=mender,number=864,comment={"body":"Hi '
        ":smiley_cat:\\nI did my very best, and this is the result of the cherry pick "
        "operation:\\n* 3.1.x :heavy_check_mark: #0\\n* 3.0.x "
        ':heavy_check_mark: #0\\n* 2.6.x :heavy_check_mark: #0\\n"}',
    ]
