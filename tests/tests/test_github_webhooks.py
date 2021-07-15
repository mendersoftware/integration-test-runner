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
        "info:create-artifact-worker version master is being used in master",
        "info:deployments version master is being used in master",
        "info:deployments-enterprise version master is being used in master",
        "info:deviceauth version master is being used in master",
        "info:deviceconfig version master is being used in master",
        "info:deviceconnect version master is being used in master",
        "info:gui version master is being used in master",
        "info:inventory version master is being used in master",
        "info:inventory-enterprise version master is being used in master",
        "info:mender version master is being used in master",
        "info:mender-artifact version master is being used in master",
        "info:mender-cli version master is being used in master",
        "info:mender-connect version master is being used in master",
        "info:mtls-ambassador version master is being used in master",
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


def test_push(integration_test_runner_url):
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
        "info:create-artifact-worker version master is being used in master",
        "info:deployments version master is being used in master",
        "info:deployments-enterprise version master is being used in master",
        "info:deviceauth version master is being used in master",
        "info:deviceconfig version master is being used in master",
        "info:gui version master is being used in master",
        "info:inventory version master is being used in master",
        "info:inventory-enterprise version master is being used in master",
        "info:mender version master is being used in master",
        "info:mender-artifact version master is being used in master",
        "info:mender-cli version master is being used in master",
        "info:mender-connect version master is being used in master",
        "info:mtls-ambassador version master is being used in master",
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
        "variables: AUDITLOGS_REV:master, BUILD_BEAGLEBONEBLACK:, BUILD_CLIENT:false, "
        "BUILD_QEMUX86_64_BIOS_GRUB:, BUILD_QEMUX86_64_BIOS_GRUB_GPT:, "
        "BUILD_QEMUX86_64_UEFI_GRUB:, BUILD_VEXPRESS_QEMU:, "
        "BUILD_VEXPRESS_QEMU_FLASH:, BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, "
        "CREATE_ARTIFACT_WORKER_REV:master, DEPLOYMENTS_ENTERPRISE_REV:master, "
        "DEPLOYMENTS_REV:master, DEVICEAUTH_REV:master, DEVICECONFIG_REV:master, "
        "DEVICECONNECT_REV:pull/109/head, GUI_REV:master, INTEGRATION_REV:master, "
        "INVENTORY_ENTERPRISE_REV:master, INVENTORY_REV:master, "
        "MENDER_ARTIFACT_REV:master, MENDER_CLI_REV:master, "
        "MENDER_CONNECT_REV:master, MENDER_REV:master, MTLS_AMBASSADOR_REV:master, "
        "RUN_INTEGRATION_TESTS:true, TENANTADM_REV:master, "
        "TEST_QEMUX86_64_BIOS_GRUB:, TEST_QEMUX86_64_BIOS_GRUB_GPT:, "
        "TEST_QEMUX86_64_UEFI_GRUB:, TEST_VEXPRESS_QEMU:, TEST_VEXPRESS_QEMU_FLASH:, "
        "TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, USERADM_ENTERPRISE_REV:master, "
        "USERADM_REV:master, WORKFLOWS_ENTERPRISE_REV:master, WORKFLOWS_REV:master, ",
        "gitlab.CreatePipeline: "
        'path=Northern.tech/Mender/mender-qa,options={"ref":"master","variables":[{"key":"AUDITLOGS_REV","value":"master"},{"key":"BUILD_BEAGLEBONEBLACK","value":""},{"key":"BUILD_CLIENT","value":"false"},{"key":"BUILD_QEMUX86_64_BIOS_GRUB","value":""},{"key":"BUILD_QEMUX86_64_BIOS_GRUB_GPT","value":""},{"key":"BUILD_QEMUX86_64_UEFI_GRUB","value":""},{"key":"BUILD_VEXPRESS_QEMU","value":""},{"key":"BUILD_VEXPRESS_QEMU_FLASH","value":""},{"key":"BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},{"key":"CREATE_ARTIFACT_WORKER_REV","value":"master"},{"key":"DEPLOYMENTS_ENTERPRISE_REV","value":"master"},{"key":"DEPLOYMENTS_REV","value":"master"},{"key":"DEVICEAUTH_REV","value":"master"},{"key":"DEVICECONFIG_REV","value":"master"},{"key":"DEVICECONNECT_REV","value":"pull/109/head"},{"key":"GUI_REV","value":"master"},{"key":"INTEGRATION_REV","value":"master"},{"key":"INVENTORY_ENTERPRISE_REV","value":"master"},{"key":"INVENTORY_REV","value":"master"},{"key":"MENDER_ARTIFACT_REV","value":"master"},{"key":"MENDER_CLI_REV","value":"master"},{"key":"MENDER_CONNECT_REV","value":"master"},{"key":"MENDER_REV","value":"master"},{"key":"MTLS_AMBASSADOR_REV","value":"master"},{"key":"RUN_INTEGRATION_TESTS","value":"true"},{"key":"TENANTADM_REV","value":"master"},{"key":"TEST_QEMUX86_64_BIOS_GRUB","value":""},{"key":"TEST_QEMUX86_64_BIOS_GRUB_GPT","value":""},{"key":"TEST_QEMUX86_64_UEFI_GRUB","value":""},{"key":"TEST_VEXPRESS_QEMU","value":""},{"key":"TEST_VEXPRESS_QEMU_FLASH","value":""},{"key":"TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},{"key":"USERADM_ENTERPRISE_REV","value":"master"},{"key":"USERADM_REV","value":"master"},{"key":"WORKFLOWS_ENTERPRISE_REV","value":"master"},{"key":"WORKFLOWS_REV","value":"master"}]}',
        "info:Created pipeline: ",
        "github.CreateComment: "
        'org=mendersoftware,repo=deviceconnect,number=109,comment={"body":"\\nHello '
        ":smile_cat: I created a pipeline for you here: "
        "[Pipeline-0]()\\n\\n\\u003cdetails\\u003e\\n    \\u003csummary\\u003eBuild "
        "Configuration Matrix\\u003c/summary\\u003e\\u003cp\\u003e\\n\\n| Key   | "
        "Value |\\n| ----- | ----- |\\n| AUDITLOGS_REV | master |\\n| BUILD_CLIENT | "
        "false |\\n| CREATE_ARTIFACT_WORKER_REV | master |\\n| "
        "DEPLOYMENTS_ENTERPRISE_REV | master |\\n| DEPLOYMENTS_REV | master |\\n| "
        "DEVICEAUTH_REV | master |\\n| DEVICECONFIG_REV | master |\\n| "
        "DEVICECONNECT_REV | pull/109/head |\\n| GUI_REV | master |\\n| "
        "INTEGRATION_REV | master |\\n| INVENTORY_ENTERPRISE_REV | master |\\n| "
        "INVENTORY_REV | master |\\n| MENDER_ARTIFACT_REV | master |\\n| "
        "MENDER_CLI_REV | master |\\n| MENDER_CONNECT_REV | master |\\n| MENDER_REV | "
        "master |\\n| MTLS_AMBASSADOR_REV | master |\\n| RUN_INTEGRATION_TESTS | true "
        "|\\n| TENANTADM_REV | master |\\n| USERADM_ENTERPRISE_REV | master |\\n| "
        "USERADM_REV | master |\\n| WORKFLOWS_ENTERPRISE_REV | master |\\n| "
        "WORKFLOWS_REV | master |\\n\\n\\n "
        '\\u003c/p\\u003e\\u003c/details\\u003e\\n"}',
    ]
