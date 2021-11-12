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

import pytest

BASE_DIR = os.path.dirname(__file__)


def load_payload(filename):
    with open(os.path.join(BASE_DIR, "payloads", filename), "rb") as f:
        return f.read()


@pytest.mark.golden_test("golden-files/test_pull_request_opened_from_fork.yml")
def test_pull_request_opened_from_fork(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_pull_request_opened_from_branch.yml")
def test_pull_request_opened_from_branch(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]



@pytest.mark.golden_test("golden-files/test_pull_request_closed.yml")
def test_pull_request_closed(golden,integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_push_mendersoftware.yml")
def test_push_mendersoftware(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_push_cfengine.yml")
def test_push_cfengine(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_push_mender_qa_repo.yml")
def test_push_mender_qa_repo(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_issue_comment.yml")
def test_issue_comment(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_issue_comment_minor_series.yml")
def test_issue_comment_minor_series(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]


@pytest.mark.golden_test("golden-files/test_cherrypick.yml")
def test_cherrypick(golden, integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload(golden["input"]),
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
    assert res.json() == golden.out["output"]
