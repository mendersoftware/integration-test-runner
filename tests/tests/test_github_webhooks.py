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


def test_pull_request_opened(integration_test_runner_url):
    res = requests.post(
        integration_test_runner_url + "/",
        data=load_payload("pull_request_opened.json"),
        headers={
            "Content-Type": "application/json",
            "X-Github-Event": "pull_request",
            "X-Github-Delivery": "delivery",
        },
    )
    assert res.status_code == 200
    #
    res = requests.get(integration_test_runner_url + "/logs",)
    assert res.status_code == 200
    assert res.json() == [
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "git.Run: /usr/bin/git push -f --set-upstream gitlab pr_140",
        "info:Created branch: workflows:pr_140",
        "info:Pipeline is expected to start automatically",
        "debug:deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline",
        "info:Ignoring cherry-pick suggestions for action: opened, merged: false",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:workflows:140 would trigger 1 builds",
        "info:I have already commented on the pr: workflows/140, no need to keep on nagging",
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
    assert res.status_code == 200
    #
    res = requests.get(integration_test_runner_url + "/logs",)
    assert res.status_code == 200
    assert res.json() == [
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "git.Run: /usr/bin/git push -f --set-upstream gitlab pr_140",
        "info:Created branch: workflows:pr_140",
        "info:Pipeline is expected to start automatically",
        "debug:deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline",
        "info:Ignoring cherry-pick suggestions for action: opened, merged: false",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:workflows:140 would trigger 1 builds",
        "info:I have already commented on the pr: workflows/140, no need to keep on nagging",
        "info:createPullRequestBranch: Action closed, ignoring",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch gitlab",
        "git.Run: /usr/bin/git push gitlab --delete pr_140",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git fetch github master:local",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "info:Found no changelog entries, ignoring cherry-pick suggestions",
        "debug:stopBuildsOfStalePRs: Find any running pipelines and kill mercilessly!",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:auditlogs version origin/master is being used in master",
        "info:create-artifact-worker version origin/master is being used in master",
        "info:deployments version origin/master is being used in master",
        "info:deployments-enterprise version origin/master is being used in master",
        "info:deviceauth version origin/master is being used in master",
        "info:deviceconfig version origin/master is being used in master",
        "info:deviceconnect version origin/master is being used in master",
        "info:gui version origin/master is being used in master",
        "info:inventory version origin/master is being used in master",
        "info:inventory-enterprise version origin/master is being used in master",
        "info:mender version origin/master is being used in master",
        "info:mender-artifact version origin/master is being used in master",
        "info:mender-cli version origin/master is being used in master",
        "info:mender-connect version origin/master is being used in master",
        "info:mtls-ambassador version origin/master is being used in master",
        "info:tenantadm version origin/master is being used in master",
        "info:useradm version origin/master is being used in master",
        "info:useradm-enterprise version origin/master is being used in master",
        "info:workflows-enterprise version origin/master is being used in master",
        "info:syncIfOSHasEnterpriseRepo: Merge to (master) in an OS repository detected. Syncing the repositories...",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add opensource git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add enterprise git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add mender-test-bot git@github.com:/mender-test-bot/workflows-enterprise.git",
        "git.Run: /usr/bin/git config --add user.name mender-test-bot",
        "git.Run: /usr/bin/git config --add user.email mender@northern.tech",
        "git.Run: /usr/bin/git fetch opensource master",
        "git.Run: /usr/bin/git fetch enterprise master:mergeostoent_140",
        "git.Run: /usr/bin/git checkout mergeostoent_140",
        "debug:Trying to Merge OS base branch: (master) including PR: (140) into Enterprise: (master)",
        "git.Run: /usr/bin/git merge -m Merge OS base branch: (master) including PR: (140) into Enterprise: (master) opensource/master",
        "git.Run: /usr/bin/git push --set-upstream mender-test-bot mergeostoent_140",
        "info:Merged branch: opensource/workflows/master into enterprise/workflows/master in the Enterprise repo",
        'github.CreatePullRequest: org=mendersoftware,repo=workflows-enterprise,pr={"title":"[Bot] Improve logging","head":"mender-test-bot:mergeostoent_140","base":"master","body":"Original PR: https://github.com/mendersoftware/workflows/pull/140\\n\\nChangelog: none\\r\\n\\r\\nSigned-off-by: Fabio Tranchitella \\u003cfabio.tranchitella@northern.tech\\u003e","maintainer_can_modify":true}',
        "info:syncIfOSHasEnterpriseRepo: Created PR: 0 on Enterprise/workflows/master",
        'debug:syncIfOSHasEnterpriseRepo: Created PR: github.PullRequest{ID:666510619, Number:140, State:"closed", Locked:false, Title:"Improve logging", Body:"Changelog: none\r\n\r\nSigned-off-by: Fabio Tranchitella <fabio.tranchitella@northern.tech>", CreatedAt:time.Time{wall:, ext:}, UpdatedAt:time.Time{wall:, ext:}, ClosedAt:time.Time{wall:, ext:}, MergedAt:time.Time{wall:, ext:}, Labels:[], User:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, Draft:false, Merged:true, MergeableState:"unknown", MergedBy:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, MergeCommitSHA:"9a296d956f3deba8abd404ee49e68c1c19ea18b5", Comments:3, Commits:1, Additions:15, Deletions:7, ChangedFiles:2, URL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140", HTMLURL:"https://github.com/mendersoftware/workflows/pull/140", IssueURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140", StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9", DiffURL:"https://github.com/mendersoftware/workflows/pull/140.diff", PatchURL:"https://github.com/mendersoftware/workflows/pull/140.patch", CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits", CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments", ReviewCommentsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments", ReviewCommentURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}", ReviewComments:0, Assignees:[], MaintainerCanModify:false, AuthorAssociation:"CONTRIBUTOR", NodeID:"MDExOlB1bGxSZXF1ZXN0NjY2NTEwNjE5", RequestedReviewers:[], RequestedTeams:[], Links:github.PRLinks{Self:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140"}, HTML:github.PRLink{HRef:"https://github.com/mendersoftware/workflows/pull/140"}, Issue:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140"}, Comments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments"}, ReviewComments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments"}, ReviewComment:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}"}, Commits:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits"}, Statuses:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9"}}, Head:github.PullRequestBranch{Label:"tranchitella:men-4705", Ref:"men-4705", SHA:"7b099b84cb50df18847027b0afa16820eab850d9", Repo:github.Repository{ID:229675849, NodeID:"MDEwOlJlcG9zaXRvcnkyMjk2NzU4NDk=", Owner:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, Name:"workflows", FullName:"tranchitella/workflows", Description:"Workflow orchestrator for Mender", DefaultBranch:"master", CreatedAt:github.Timestamp{2019-12-23 04:24:26 +0000 UTC}, PushedAt:github.Timestamp{2021-06-10 05:06:50 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-09 04:25:50 +0000 UTC}, HTMLURL:"https://github.com/tranchitella/workflows", CloneURL:"https://github.com/tranchitella/workflows.git", GitURL:"git://github.com/tranchitella/workflows.git", SSHURL:"git@github.com:tranchitella/workflows.git", SVNURL:"https://github.com/tranchitella/workflows", Language:"Go", Fork:true, ForksCount:0, OpenIssuesCount:0, StargazersCount:0, WatchersCount:0, Size:5656, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, Private:false, HasIssues:false, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, URL:"https://api.github.com/repos/tranchitella/workflows", ArchiveURL:"https://api.github.com/repos/tranchitella/workflows/{archive_format}{/ref}", AssigneesURL:"https://api.github.com/repos/tranchitella/workflows/assignees{/user}", BlobsURL:"https://api.github.com/repos/tranchitella/workflows/git/blobs{/sha}", BranchesURL:"https://api.github.com/repos/tranchitella/workflows/branches{/branch}", CollaboratorsURL:"https://api.github.com/repos/tranchitella/workflows/collaborators{/collaborator}", CommentsURL:"https://api.github.com/repos/tranchitella/workflows/comments{/number}", CommitsURL:"https://api.github.com/repos/tranchitella/workflows/commits{/sha}", CompareURL:"https://api.github.com/repos/tranchitella/workflows/compare/{base}...{head}", ContentsURL:"https://api.github.com/repos/tranchitella/workflows/contents/{+path}", ContributorsURL:"https://api.github.com/repos/tranchitella/workflows/contributors", DeploymentsURL:"https://api.github.com/repos/tranchitella/workflows/deployments", DownloadsURL:"https://api.github.com/repos/tranchitella/workflows/downloads", EventsURL:"https://api.github.com/repos/tranchitella/workflows/events", ForksURL:"https://api.github.com/repos/tranchitella/workflows/forks", GitCommitsURL:"https://api.github.com/repos/tranchitella/workflows/git/commits{/sha}", GitRefsURL:"https://api.github.com/repos/tranchitella/workflows/git/refs{/sha}", GitTagsURL:"https://api.github.com/repos/tranchitella/workflows/git/tags{/sha}", HooksURL:"https://api.github.com/repos/tranchitella/workflows/hooks", IssueCommentURL:"https://api.github.com/repos/tranchitella/workflows/issues/comments{/number}", IssueEventsURL:"https://api.github.com/repos/tranchitella/workflows/issues/events{/number}", IssuesURL:"https://api.github.com/repos/tranchitella/workflows/issues{/number}", KeysURL:"https://api.github.com/repos/tranchitella/workflows/keys{/key_id}", LabelsURL:"https://api.github.com/repos/tranchitella/workflows/labels{/name}", LanguagesURL:"https://api.github.com/repos/tranchitella/workflows/languages", MergesURL:"https://api.github.com/repos/tranchitella/workflows/merges", MilestonesURL:"https://api.github.com/repos/tranchitella/workflows/milestones{/number}", NotificationsURL:"https://api.github.com/repos/tranchitella/workflows/notifications{?since,all,participating}", PullsURL:"https://api.github.com/repos/tranchitella/workflows/pulls{/number}", ReleasesURL:"https://api.github.com/repos/tranchitella/workflows/releases{/id}", StargazersURL:"https://api.github.com/repos/tranchitella/workflows/stargazers", StatusesURL:"https://api.github.com/repos/tranchitella/workflows/statuses/{sha}", SubscribersURL:"https://api.github.com/repos/tranchitella/workflows/subscribers", SubscriptionURL:"https://api.github.com/repos/tranchitella/workflows/subscription", TagsURL:"https://api.github.com/repos/tranchitella/workflows/tags", TreesURL:"https://api.github.com/repos/tranchitella/workflows/git/trees{/sha}", TeamsURL:"https://api.github.com/repos/tranchitella/workflows/teams"}, User:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}}, Base:github.PullRequestBranch{Label:"mendersoftware:master", Ref:"master", SHA:"70ab90b3932d3d008ebee56d6cfe4f3329d5ee7b", Repo:github.Repository{ID:227348934, NodeID:"MDEwOlJlcG9zaXRvcnkyMjczNDg5MzQ=", Owner:github.User{Login:"mendersoftware", ID:15040539, NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", HTMLURL:"https://github.com/mendersoftware", GravatarID:"", Type:"Organization", SiteAdmin:false, URL:"https://api.github.com/users/mendersoftware", EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", FollowersURL:"https://api.github.com/users/mendersoftware/followers", GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", ReposURL:"https://api.github.com/users/mendersoftware/repos", StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}, Name:"workflows", FullName:"mendersoftware/workflows", Description:"Workflow orchestrator for Mender", Homepage:"http://mender.io", DefaultBranch:"master", CreatedAt:github.Timestamp{2019-12-11 11:23:32 +0000 UTC}, PushedAt:github.Timestamp{2021-06-10 07:56:10 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-07 10:50:40 +0000 UTC}, HTMLURL:"https://github.com/mendersoftware/workflows", CloneURL:"https://github.com/mendersoftware/workflows.git", GitURL:"git://github.com/mendersoftware/workflows.git", SSHURL:"git@github.com:mendersoftware/workflows.git", SVNURL:"https://github.com/mendersoftware/workflows", Language:"Go", Fork:false, ForksCount:11, OpenIssuesCount:0, StargazersCount:3, WatchersCount:3, Size:5671, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, Private:false, HasIssues:true, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, URL:"https://api.github.com/repos/mendersoftware/workflows", ArchiveURL:"https://api.github.com/repos/mendersoftware/workflows/{archive_format}{/ref}", AssigneesURL:"https://api.github.com/repos/mendersoftware/workflows/assignees{/user}", BlobsURL:"https://api.github.com/repos/mendersoftware/workflows/git/blobs{/sha}", BranchesURL:"https://api.github.com/repos/mendersoftware/workflows/branches{/branch}", CollaboratorsURL:"https://api.github.com/repos/mendersoftware/workflows/collaborators{/collaborator}", CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/comments{/number}", CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/commits{/sha}", CompareURL:"https://api.github.com/repos/mendersoftware/workflows/compare/{base}...{head}", ContentsURL:"https://api.github.com/repos/mendersoftware/workflows/contents/{+path}", ContributorsURL:"https://api.github.com/repos/mendersoftware/workflows/contributors", DeploymentsURL:"https://api.github.com/repos/mendersoftware/workflows/deployments", DownloadsURL:"https://api.github.com/repos/mendersoftware/workflows/downloads", EventsURL:"https://api.github.com/repos/mendersoftware/workflows/events", ForksURL:"https://api.github.com/repos/mendersoftware/workflows/forks", GitCommitsURL:"https://api.github.com/repos/mendersoftware/workflows/git/commits{/sha}", GitRefsURL:"https://api.github.com/repos/mendersoftware/workflows/git/refs{/sha}", GitTagsURL:"https://api.github.com/repos/mendersoftware/workflows/git/tags{/sha}", HooksURL:"https://api.github.com/repos/mendersoftware/workflows/hooks", IssueCommentURL:"https://api.github.com/repos/mendersoftware/workflows/issues/comments{/number}", IssueEventsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/events{/number}", IssuesURL:"https://api.github.com/repos/mendersoftware/workflows/issues{/number}", KeysURL:"https://api.github.com/repos/mendersoftware/workflows/keys{/key_id}", LabelsURL:"https://api.github.com/repos/mendersoftware/workflows/labels{/name}", LanguagesURL:"https://api.github.com/repos/mendersoftware/workflows/languages", MergesURL:"https://api.github.com/repos/mendersoftware/workflows/merges", MilestonesURL:"https://api.github.com/repos/mendersoftware/workflows/milestones{/number}", NotificationsURL:"https://api.github.com/repos/mendersoftware/workflows/notifications{?since,all,participating}", PullsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls{/number}", ReleasesURL:"https://api.github.com/repos/mendersoftware/workflows/releases{/id}", StargazersURL:"https://api.github.com/repos/mendersoftware/workflows/stargazers", StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/{sha}", SubscribersURL:"https://api.github.com/repos/mendersoftware/workflows/subscribers", SubscriptionURL:"https://api.github.com/repos/mendersoftware/workflows/subscription", TagsURL:"https://api.github.com/repos/mendersoftware/workflows/tags", TreesURL:"https://api.github.com/repos/mendersoftware/workflows/git/trees{/sha}", TeamsURL:"https://api.github.com/repos/mendersoftware/workflows/teams"}, User:github.User{Login:"mendersoftware", ID:15040539, NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", HTMLURL:"https://github.com/mendersoftware", GravatarID:"", Type:"Organization", SiteAdmin:false, URL:"https://api.github.com/users/mendersoftware", EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", FollowersURL:"https://api.github.com/users/mendersoftware/followers", GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", ReposURL:"https://api.github.com/users/mendersoftware/repos", StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}}}',
        "debug:Trying to @mention the user in the newly created PR",
        "debug:userName: tranchitella",
        'github.CreateComment: org=mendersoftware,repo=workflows-enterprise,number=0,comment={"body":"@tranchitella I have created a PR for you, ready to merge as soon as tests are passed"}',
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
    assert res.status_code == 200
    #
    res = requests.get(integration_test_runner_url + "/logs",)
    assert res.status_code == 200
    assert res.json() == [
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "git.Run: /usr/bin/git push -f --set-upstream gitlab pr_140",
        "info:Created branch: workflows:pr_140",
        "info:Pipeline is expected to start automatically",
        "debug:deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline",
        "info:Ignoring cherry-pick suggestions for action: opened, merged: false",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:workflows:140 would trigger 1 builds",
        "info:I have already commented on the pr: workflows/140, no need to keep on nagging",
        "info:createPullRequestBranch: Action closed, ignoring",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch gitlab",
        "git.Run: /usr/bin/git push gitlab --delete pr_140",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git fetch github master:local",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "info:Found no changelog entries, ignoring cherry-pick suggestions",
        "debug:stopBuildsOfStalePRs: Find any running pipelines and kill mercilessly!",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:auditlogs version origin/master is being used in master",
        "info:create-artifact-worker version origin/master is being used in master",
        "info:deployments version origin/master is being used in master",
        "info:deployments-enterprise version origin/master is being used in master",
        "info:deviceauth version origin/master is being used in master",
        "info:deviceconfig version origin/master is being used in master",
        "info:deviceconnect version origin/master is being used in master",
        "info:gui version origin/master is being used in master",
        "info:inventory version origin/master is being used in master",
        "info:inventory-enterprise version origin/master is being used in master",
        "info:mender version origin/master is being used in master",
        "info:mender-artifact version origin/master is being used in master",
        "info:mender-cli version origin/master is being used in master",
        "info:mender-connect version origin/master is being used in master",
        "info:mtls-ambassador version origin/master is being used in master",
        "info:tenantadm version origin/master is being used in master",
        "info:useradm version origin/master is being used in master",
        "info:useradm-enterprise version origin/master is being used in master",
        "info:workflows-enterprise version origin/master is being used in master",
        "info:syncIfOSHasEnterpriseRepo: Merge to (master) in an OS repository detected. Syncing the repositories...",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add opensource git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add enterprise git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add mender-test-bot git@github.com:/mender-test-bot/workflows-enterprise.git",
        "git.Run: /usr/bin/git config --add user.name mender-test-bot",
        "git.Run: /usr/bin/git config --add user.email mender@northern.tech",
        "git.Run: /usr/bin/git fetch opensource master",
        "git.Run: /usr/bin/git fetch enterprise master:mergeostoent_140",
        "git.Run: /usr/bin/git checkout mergeostoent_140",
        "debug:Trying to Merge OS base branch: (master) including PR: (140) into Enterprise: (master)",
        "git.Run: /usr/bin/git merge -m Merge OS base branch: (master) including PR: (140) into Enterprise: (master) opensource/master",
        "git.Run: /usr/bin/git push --set-upstream mender-test-bot mergeostoent_140",
        "info:Merged branch: opensource/workflows/master into enterprise/workflows/master in the Enterprise repo",
        'github.CreatePullRequest: org=mendersoftware,repo=workflows-enterprise,pr={"title":"[Bot] Improve logging","head":"mender-test-bot:mergeostoent_140","base":"master","body":"Original PR: https://github.com/mendersoftware/workflows/pull/140\\n\\nChangelog: none\\r\\n\\r\\nSigned-off-by: Fabio Tranchitella \\u003cfabio.tranchitella@northern.tech\\u003e","maintainer_can_modify":true}',
        "info:syncIfOSHasEnterpriseRepo: Created PR: 0 on Enterprise/workflows/master",
        'debug:syncIfOSHasEnterpriseRepo: Created PR: github.PullRequest{ID:666510619, Number:140, State:"closed", Locked:false, Title:"Improve logging", Body:"Changelog: none\r\n\r\nSigned-off-by: Fabio Tranchitella <fabio.tranchitella@northern.tech>", CreatedAt:time.Time{wall:, ext:}, UpdatedAt:time.Time{wall:, ext:}, ClosedAt:time.Time{wall:, ext:}, MergedAt:time.Time{wall:, ext:}, Labels:[], User:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, Draft:false, Merged:true, MergeableState:"unknown", MergedBy:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, MergeCommitSHA:"9a296d956f3deba8abd404ee49e68c1c19ea18b5", Comments:3, Commits:1, Additions:15, Deletions:7, ChangedFiles:2, URL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140", HTMLURL:"https://github.com/mendersoftware/workflows/pull/140", IssueURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140", StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9", DiffURL:"https://github.com/mendersoftware/workflows/pull/140.diff", PatchURL:"https://github.com/mendersoftware/workflows/pull/140.patch", CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits", CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments", ReviewCommentsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments", ReviewCommentURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}", ReviewComments:0, Assignees:[], MaintainerCanModify:false, AuthorAssociation:"CONTRIBUTOR", NodeID:"MDExOlB1bGxSZXF1ZXN0NjY2NTEwNjE5", RequestedReviewers:[], RequestedTeams:[], Links:github.PRLinks{Self:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140"}, HTML:github.PRLink{HRef:"https://github.com/mendersoftware/workflows/pull/140"}, Issue:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140"}, Comments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments"}, ReviewComments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments"}, ReviewComment:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}"}, Commits:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits"}, Statuses:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9"}}, Head:github.PullRequestBranch{Label:"tranchitella:men-4705", Ref:"men-4705", SHA:"7b099b84cb50df18847027b0afa16820eab850d9", Repo:github.Repository{ID:229675849, NodeID:"MDEwOlJlcG9zaXRvcnkyMjk2NzU4NDk=", Owner:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, Name:"workflows", FullName:"tranchitella/workflows", Description:"Workflow orchestrator for Mender", DefaultBranch:"master", CreatedAt:github.Timestamp{2019-12-23 04:24:26 +0000 UTC}, PushedAt:github.Timestamp{2021-06-10 05:06:50 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-09 04:25:50 +0000 UTC}, HTMLURL:"https://github.com/tranchitella/workflows", CloneURL:"https://github.com/tranchitella/workflows.git", GitURL:"git://github.com/tranchitella/workflows.git", SSHURL:"git@github.com:tranchitella/workflows.git", SVNURL:"https://github.com/tranchitella/workflows", Language:"Go", Fork:true, ForksCount:0, OpenIssuesCount:0, StargazersCount:0, WatchersCount:0, Size:5656, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, Private:false, HasIssues:false, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, URL:"https://api.github.com/repos/tranchitella/workflows", ArchiveURL:"https://api.github.com/repos/tranchitella/workflows/{archive_format}{/ref}", AssigneesURL:"https://api.github.com/repos/tranchitella/workflows/assignees{/user}", BlobsURL:"https://api.github.com/repos/tranchitella/workflows/git/blobs{/sha}", BranchesURL:"https://api.github.com/repos/tranchitella/workflows/branches{/branch}", CollaboratorsURL:"https://api.github.com/repos/tranchitella/workflows/collaborators{/collaborator}", CommentsURL:"https://api.github.com/repos/tranchitella/workflows/comments{/number}", CommitsURL:"https://api.github.com/repos/tranchitella/workflows/commits{/sha}", CompareURL:"https://api.github.com/repos/tranchitella/workflows/compare/{base}...{head}", ContentsURL:"https://api.github.com/repos/tranchitella/workflows/contents/{+path}", ContributorsURL:"https://api.github.com/repos/tranchitella/workflows/contributors", DeploymentsURL:"https://api.github.com/repos/tranchitella/workflows/deployments", DownloadsURL:"https://api.github.com/repos/tranchitella/workflows/downloads", EventsURL:"https://api.github.com/repos/tranchitella/workflows/events", ForksURL:"https://api.github.com/repos/tranchitella/workflows/forks", GitCommitsURL:"https://api.github.com/repos/tranchitella/workflows/git/commits{/sha}", GitRefsURL:"https://api.github.com/repos/tranchitella/workflows/git/refs{/sha}", GitTagsURL:"https://api.github.com/repos/tranchitella/workflows/git/tags{/sha}", HooksURL:"https://api.github.com/repos/tranchitella/workflows/hooks", IssueCommentURL:"https://api.github.com/repos/tranchitella/workflows/issues/comments{/number}", IssueEventsURL:"https://api.github.com/repos/tranchitella/workflows/issues/events{/number}", IssuesURL:"https://api.github.com/repos/tranchitella/workflows/issues{/number}", KeysURL:"https://api.github.com/repos/tranchitella/workflows/keys{/key_id}", LabelsURL:"https://api.github.com/repos/tranchitella/workflows/labels{/name}", LanguagesURL:"https://api.github.com/repos/tranchitella/workflows/languages", MergesURL:"https://api.github.com/repos/tranchitella/workflows/merges", MilestonesURL:"https://api.github.com/repos/tranchitella/workflows/milestones{/number}", NotificationsURL:"https://api.github.com/repos/tranchitella/workflows/notifications{?since,all,participating}", PullsURL:"https://api.github.com/repos/tranchitella/workflows/pulls{/number}", ReleasesURL:"https://api.github.com/repos/tranchitella/workflows/releases{/id}", StargazersURL:"https://api.github.com/repos/tranchitella/workflows/stargazers", StatusesURL:"https://api.github.com/repos/tranchitella/workflows/statuses/{sha}", SubscribersURL:"https://api.github.com/repos/tranchitella/workflows/subscribers", SubscriptionURL:"https://api.github.com/repos/tranchitella/workflows/subscription", TagsURL:"https://api.github.com/repos/tranchitella/workflows/tags", TreesURL:"https://api.github.com/repos/tranchitella/workflows/git/trees{/sha}", TeamsURL:"https://api.github.com/repos/tranchitella/workflows/teams"}, User:github.User{Login:"tranchitella", ID:1295287, NodeID:"MDQ6VXNlcjEyOTUyODc=", AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", FollowersURL:"https://api.github.com/users/tranchitella/followers", GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", ReposURL:"https://api.github.com/users/tranchitella/repos", StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}}, Base:github.PullRequestBranch{Label:"mendersoftware:master", Ref:"master", SHA:"70ab90b3932d3d008ebee56d6cfe4f3329d5ee7b", Repo:github.Repository{ID:227348934, NodeID:"MDEwOlJlcG9zaXRvcnkyMjczNDg5MzQ=", Owner:github.User{Login:"mendersoftware", ID:15040539, NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", HTMLURL:"https://github.com/mendersoftware", GravatarID:"", Type:"Organization", SiteAdmin:false, URL:"https://api.github.com/users/mendersoftware", EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", FollowersURL:"https://api.github.com/users/mendersoftware/followers", GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", ReposURL:"https://api.github.com/users/mendersoftware/repos", StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}, Name:"workflows", FullName:"mendersoftware/workflows", Description:"Workflow orchestrator for Mender", Homepage:"http://mender.io", DefaultBranch:"master", CreatedAt:github.Timestamp{2019-12-11 11:23:32 +0000 UTC}, PushedAt:github.Timestamp{2021-06-10 07:56:10 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-07 10:50:40 +0000 UTC}, HTMLURL:"https://github.com/mendersoftware/workflows", CloneURL:"https://github.com/mendersoftware/workflows.git", GitURL:"git://github.com/mendersoftware/workflows.git", SSHURL:"git@github.com:mendersoftware/workflows.git", SVNURL:"https://github.com/mendersoftware/workflows", Language:"Go", Fork:false, ForksCount:11, OpenIssuesCount:0, StargazersCount:3, WatchersCount:3, Size:5671, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, Private:false, HasIssues:true, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, URL:"https://api.github.com/repos/mendersoftware/workflows", ArchiveURL:"https://api.github.com/repos/mendersoftware/workflows/{archive_format}{/ref}", AssigneesURL:"https://api.github.com/repos/mendersoftware/workflows/assignees{/user}", BlobsURL:"https://api.github.com/repos/mendersoftware/workflows/git/blobs{/sha}", BranchesURL:"https://api.github.com/repos/mendersoftware/workflows/branches{/branch}", CollaboratorsURL:"https://api.github.com/repos/mendersoftware/workflows/collaborators{/collaborator}", CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/comments{/number}", CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/commits{/sha}", CompareURL:"https://api.github.com/repos/mendersoftware/workflows/compare/{base}...{head}", ContentsURL:"https://api.github.com/repos/mendersoftware/workflows/contents/{+path}", ContributorsURL:"https://api.github.com/repos/mendersoftware/workflows/contributors", DeploymentsURL:"https://api.github.com/repos/mendersoftware/workflows/deployments", DownloadsURL:"https://api.github.com/repos/mendersoftware/workflows/downloads", EventsURL:"https://api.github.com/repos/mendersoftware/workflows/events", ForksURL:"https://api.github.com/repos/mendersoftware/workflows/forks", GitCommitsURL:"https://api.github.com/repos/mendersoftware/workflows/git/commits{/sha}", GitRefsURL:"https://api.github.com/repos/mendersoftware/workflows/git/refs{/sha}", GitTagsURL:"https://api.github.com/repos/mendersoftware/workflows/git/tags{/sha}", HooksURL:"https://api.github.com/repos/mendersoftware/workflows/hooks", IssueCommentURL:"https://api.github.com/repos/mendersoftware/workflows/issues/comments{/number}", IssueEventsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/events{/number}", IssuesURL:"https://api.github.com/repos/mendersoftware/workflows/issues{/number}", KeysURL:"https://api.github.com/repos/mendersoftware/workflows/keys{/key_id}", LabelsURL:"https://api.github.com/repos/mendersoftware/workflows/labels{/name}", LanguagesURL:"https://api.github.com/repos/mendersoftware/workflows/languages", MergesURL:"https://api.github.com/repos/mendersoftware/workflows/merges", MilestonesURL:"https://api.github.com/repos/mendersoftware/workflows/milestones{/number}", NotificationsURL:"https://api.github.com/repos/mendersoftware/workflows/notifications{?since,all,participating}", PullsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls{/number}", ReleasesURL:"https://api.github.com/repos/mendersoftware/workflows/releases{/id}", StargazersURL:"https://api.github.com/repos/mendersoftware/workflows/stargazers", StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/{sha}", SubscribersURL:"https://api.github.com/repos/mendersoftware/workflows/subscribers", SubscriptionURL:"https://api.github.com/repos/mendersoftware/workflows/subscription", TagsURL:"https://api.github.com/repos/mendersoftware/workflows/tags", TreesURL:"https://api.github.com/repos/mendersoftware/workflows/git/trees{/sha}", TeamsURL:"https://api.github.com/repos/mendersoftware/workflows/teams"}, User:github.User{Login:"mendersoftware", ID:15040539, NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", HTMLURL:"https://github.com/mendersoftware", GravatarID:"", Type:"Organization", SiteAdmin:false, URL:"https://api.github.com/users/mendersoftware", EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", FollowersURL:"https://api.github.com/users/mendersoftware/followers", GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", ReposURL:"https://api.github.com/users/mendersoftware/repos", StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}}}',
        "debug:Trying to @mention the user in the newly created PR",
        "debug:userName: tranchitella",
        'github.CreateComment: org=mendersoftware,repo=workflows-enterprise,number=0,comment={"body":"@tranchitella I have created a PR for you, ready to merge as soon as tests are passed"}',
        "info:Pull request event with action: closed",
        "info:workflows:140 would trigger 0 builds",
        "debug:Got push event :: repo workflows-enterprise :: ref refs/heads/master",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows-enterprise",
        "git.Run: /usr/bin/git fetch github",
        "git.Run: /usr/bin/git checkout -b master github/master",
        "git.Run: /usr/bin/git push -f gitlab master",
        "info:Pushed ref to GitLab: workflows-enterprise:refs/heads/master",
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
    assert res.status_code == 200
    #
    res = requests.get(integration_test_runner_url + "/logs",)
    assert res.status_code == 200
    assert res.json() == [
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "git.Run: /usr/bin/git push -f --set-upstream gitlab pr_140",
        "info:Created branch: workflows:pr_140",
        "info:Pipeline is expected to start automatically",
        "debug:deleteStaleGitlabPRBranch: PR not closed, therefore not stopping it's pipeline",
        "info:Ignoring cherry-pick suggestions for action: opened, merged: false",
        "debug:stopBuildsOfStalePRs: PR not closed, therefore not stopping it's pipeline",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:workflows:140 would trigger 1 builds",
        "info:I have already commented on the pr: workflows/140, no need to keep on nagging",
        "info:createPullRequestBranch: Action closed, ignoring",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows",
        "git.Run: /usr/bin/git fetch gitlab",
        "git.Run: /usr/bin/git push gitlab --delete pr_140",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git fetch github master:local",
        "git.Run: /usr/bin/git fetch github pull/140/head:pr_140",
        "info:Found no changelog entries, ignoring cherry-pick suggestions",
        "debug:stopBuildsOfStalePRs: Find any running pipelines and kill mercilessly!",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:workflows/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using workflows/master",
        "info:auditlogs version origin/master is being used in master",
        "info:create-artifact-worker version origin/master is being used in master",
        "info:deployments version origin/master is being used in master",
        "info:deployments-enterprise version origin/master is being used in master",
        "info:deviceauth version origin/master is being used in master",
        "info:deviceconfig version origin/master is being used in master",
        "info:deviceconnect version origin/master is being used in master",
        "info:gui version origin/master is being used in master",
        "info:inventory version origin/master is being used in master",
        "info:inventory-enterprise version origin/master is being used in master",
        "info:mender version origin/master is being used in master",
        "info:mender-artifact version origin/master is being used in master",
        "info:mender-cli version origin/master is being used in master",
        "info:mender-connect version origin/master is being used in master",
        "info:mtls-ambassador version origin/master is being used in master",
        "info:tenantadm version origin/master is being used in master",
        "info:useradm version origin/master is being used in master",
        "info:useradm-enterprise version origin/master is being used in master",
        "info:workflows-enterprise version origin/master is being used in master",
        "info:syncIfOSHasEnterpriseRepo: Merge to (master) in an OS repository detected. Syncing the repositories...",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add opensource git@github.com:/mendersoftware/workflows.git",
        "git.Run: /usr/bin/git remote add enterprise git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add mender-test-bot git@github.com:/mender-test-bot/workflows-enterprise.git",
        "git.Run: /usr/bin/git config --add user.name mender-test-bot",
        "git.Run: /usr/bin/git config --add user.email mender@northern.tech",
        "git.Run: /usr/bin/git fetch opensource master",
        "git.Run: /usr/bin/git fetch enterprise master:mergeostoent_140",
        "git.Run: /usr/bin/git checkout mergeostoent_140",
        "debug:Trying to Merge OS base branch: (master) including PR: (140) into Enterprise: (master)",
        "git.Run: /usr/bin/git merge -m Merge OS base branch: (master) including PR: (140) into Enterprise: (master) opensource/master",
        "git.Run: /usr/bin/git push --set-upstream mender-test-bot mergeostoent_140",
        "info:Merged branch: opensource/workflows/master into enterprise/workflows/master in the Enterprise repo",
        "github.CreatePullRequest: "
        'org=mendersoftware,repo=workflows-enterprise,pr={"title":"[Bot] Improve '
        'logging","head":"mender-test-bot:mergeostoent_140","base":"master","body":"Original '
        "PR: https://github.com/mendersoftware/workflows/pull/140\\n\\nChangelog: none\\r\\n\\r\\nSigned-off-by: Fabio Tranchitella "
        '\\u003cfabio.tranchitella@northern.tech\\u003e","maintainer_can_modify":true}',
        "info:syncIfOSHasEnterpriseRepo: Created PR: 0 on Enterprise/workflows/master",
        "debug:syncIfOSHasEnterpriseRepo: Created PR: "
        'github.PullRequest{ID:666510619, Number:140, State:"closed", Locked:false, '
        'Title:"Improve logging", Body:"Changelog: none\r\n'
        "\r\n"
        'Signed-off-by: Fabio Tranchitella <fabio.tranchitella@northern.tech>", '
        "CreatedAt:time.Time{wall:, ext:}, UpdatedAt:time.Time{wall:, ext:}, ClosedAt:time.Time{wall:, ext:}, MergedAt:time.Time{wall:, ext:}, Labels:[], "
        'User:github.User{Login:"tranchitella", ID:1295287, '
        'NodeID:"MDQ6VXNlcjEyOTUyODc=", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", '
        'HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", '
        'SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", '
        'EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/tranchitella/followers", '
        'GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", '
        'ReposURL:"https://api.github.com/users/tranchitella/repos", '
        'StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, '
        'Draft:false, Merged:true, MergeableState:"unknown", '
        'MergedBy:github.User{Login:"tranchitella", ID:1295287, '
        'NodeID:"MDQ6VXNlcjEyOTUyODc=", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", '
        'HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", '
        'SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", '
        'EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/tranchitella/followers", '
        'GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", '
        'ReposURL:"https://api.github.com/users/tranchitella/repos", '
        'StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, '
        'MergeCommitSHA:"9a296d956f3deba8abd404ee49e68c1c19ea18b5", Comments:3, '
        "Commits:1, Additions:15, Deletions:7, ChangedFiles:2, "
        'URL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140", '
        'HTMLURL:"https://github.com/mendersoftware/workflows/pull/140", '
        'IssueURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140", '
        'StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9", '
        'DiffURL:"https://github.com/mendersoftware/workflows/pull/140.diff", '
        'PatchURL:"https://github.com/mendersoftware/workflows/pull/140.patch", '
        'CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits", '
        'CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments", '
        'ReviewCommentsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments", '
        'ReviewCommentURL:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}", '
        "ReviewComments:0, Assignees:[], MaintainerCanModify:false, "
        'AuthorAssociation:"CONTRIBUTOR", NodeID:"MDExOlB1bGxSZXF1ZXN0NjY2NTEwNjE5", '
        "RequestedReviewers:[], RequestedTeams:[], "
        'Links:github.PRLinks{Self:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140"}, '
        'HTML:github.PRLink{HRef:"https://github.com/mendersoftware/workflows/pull/140"}, '
        'Issue:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140"}, '
        'Comments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/issues/140/comments"}, '
        'ReviewComments:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/comments"}, '
        'ReviewComment:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/comments{/number}"}, '
        'Commits:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/pulls/140/commits"}, '
        'Statuses:github.PRLink{HRef:"https://api.github.com/repos/mendersoftware/workflows/statuses/7b099b84cb50df18847027b0afa16820eab850d9"}}, '
        'Head:github.PullRequestBranch{Label:"tranchitella:men-4705", Ref:"men-4705", '
        'SHA:"7b099b84cb50df18847027b0afa16820eab850d9", '
        "Repo:github.Repository{ID:229675849, "
        'NodeID:"MDEwOlJlcG9zaXRvcnkyMjk2NzU4NDk=", '
        'Owner:github.User{Login:"tranchitella", ID:1295287, '
        'NodeID:"MDQ6VXNlcjEyOTUyODc=", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", '
        'HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", '
        'SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", '
        'EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/tranchitella/followers", '
        'GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", '
        'ReposURL:"https://api.github.com/users/tranchitella/repos", '
        'StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}, '
        'Name:"workflows", FullName:"tranchitella/workflows", Description:"Workflow '
        'orchestrator for Mender", DefaultBranch:"master", '
        "CreatedAt:github.Timestamp{2019-12-23 04:24:26 +0000 UTC}, PushedAt:github.Timestamp{2021-06-10 05:06:50 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-09 04:25:50 +0000 UTC}, "
        'HTMLURL:"https://github.com/tranchitella/workflows", '
        'CloneURL:"https://github.com/tranchitella/workflows.git", '
        'GitURL:"git://github.com/tranchitella/workflows.git", '
        'SSHURL:"git@github.com:tranchitella/workflows.git", '
        'SVNURL:"https://github.com/tranchitella/workflows", Language:"Go", '
        "Fork:true, ForksCount:0, OpenIssuesCount:0, StargazersCount:0, WatchersCount:0, Size:5656, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, "
        'License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, '
        "Private:false, HasIssues:false, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, "
        'URL:"https://api.github.com/repos/tranchitella/workflows", '
        'ArchiveURL:"https://api.github.com/repos/tranchitella/workflows/{archive_format}{/ref}", '
        'AssigneesURL:"https://api.github.com/repos/tranchitella/workflows/assignees{/user}", '
        'BlobsURL:"https://api.github.com/repos/tranchitella/workflows/git/blobs{/sha}", '
        'BranchesURL:"https://api.github.com/repos/tranchitella/workflows/branches{/branch}", '
        'CollaboratorsURL:"https://api.github.com/repos/tranchitella/workflows/collaborators{/collaborator}", '
        'CommentsURL:"https://api.github.com/repos/tranchitella/workflows/comments{/number}", '
        'CommitsURL:"https://api.github.com/repos/tranchitella/workflows/commits{/sha}", '
        'CompareURL:"https://api.github.com/repos/tranchitella/workflows/compare/{base}...{head}", '
        'ContentsURL:"https://api.github.com/repos/tranchitella/workflows/contents/{+path}", '
        'ContributorsURL:"https://api.github.com/repos/tranchitella/workflows/contributors", '
        'DeploymentsURL:"https://api.github.com/repos/tranchitella/workflows/deployments", '
        'DownloadsURL:"https://api.github.com/repos/tranchitella/workflows/downloads", '
        'EventsURL:"https://api.github.com/repos/tranchitella/workflows/events", '
        'ForksURL:"https://api.github.com/repos/tranchitella/workflows/forks", '
        'GitCommitsURL:"https://api.github.com/repos/tranchitella/workflows/git/commits{/sha}", '
        'GitRefsURL:"https://api.github.com/repos/tranchitella/workflows/git/refs{/sha}", '
        'GitTagsURL:"https://api.github.com/repos/tranchitella/workflows/git/tags{/sha}", '
        'HooksURL:"https://api.github.com/repos/tranchitella/workflows/hooks", '
        'IssueCommentURL:"https://api.github.com/repos/tranchitella/workflows/issues/comments{/number}", '
        'IssueEventsURL:"https://api.github.com/repos/tranchitella/workflows/issues/events{/number}", '
        'IssuesURL:"https://api.github.com/repos/tranchitella/workflows/issues{/number}", '
        'KeysURL:"https://api.github.com/repos/tranchitella/workflows/keys{/key_id}", '
        'LabelsURL:"https://api.github.com/repos/tranchitella/workflows/labels{/name}", '
        'LanguagesURL:"https://api.github.com/repos/tranchitella/workflows/languages", '
        'MergesURL:"https://api.github.com/repos/tranchitella/workflows/merges", '
        'MilestonesURL:"https://api.github.com/repos/tranchitella/workflows/milestones{/number}", '
        'NotificationsURL:"https://api.github.com/repos/tranchitella/workflows/notifications{?since,all,participating}", '
        'PullsURL:"https://api.github.com/repos/tranchitella/workflows/pulls{/number}", '
        'ReleasesURL:"https://api.github.com/repos/tranchitella/workflows/releases{/id}", '
        'StargazersURL:"https://api.github.com/repos/tranchitella/workflows/stargazers", '
        'StatusesURL:"https://api.github.com/repos/tranchitella/workflows/statuses/{sha}", '
        'SubscribersURL:"https://api.github.com/repos/tranchitella/workflows/subscribers", '
        'SubscriptionURL:"https://api.github.com/repos/tranchitella/workflows/subscription", '
        'TagsURL:"https://api.github.com/repos/tranchitella/workflows/tags", '
        'TreesURL:"https://api.github.com/repos/tranchitella/workflows/git/trees{/sha}", '
        'TeamsURL:"https://api.github.com/repos/tranchitella/workflows/teams"}, '
        'User:github.User{Login:"tranchitella", ID:1295287, '
        'NodeID:"MDQ6VXNlcjEyOTUyODc=", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/1295287?v=4", '
        'HTMLURL:"https://github.com/tranchitella", GravatarID:"", Type:"User", '
        'SiteAdmin:false, URL:"https://api.github.com/users/tranchitella", '
        'EventsURL:"https://api.github.com/users/tranchitella/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/tranchitella/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/tranchitella/followers", '
        'GistsURL:"https://api.github.com/users/tranchitella/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/tranchitella/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/tranchitella/received_events", '
        'ReposURL:"https://api.github.com/users/tranchitella/repos", '
        'StarredURL:"https://api.github.com/users/tranchitella/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/tranchitella/subscriptions"}}, '
        'Base:github.PullRequestBranch{Label:"mendersoftware:master", Ref:"master", '
        'SHA:"70ab90b3932d3d008ebee56d6cfe4f3329d5ee7b", '
        "Repo:github.Repository{ID:227348934, "
        'NodeID:"MDEwOlJlcG9zaXRvcnkyMjczNDg5MzQ=", '
        'Owner:github.User{Login:"mendersoftware", ID:15040539, '
        'NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", '
        'HTMLURL:"https://github.com/mendersoftware", GravatarID:"", '
        'Type:"Organization", SiteAdmin:false, '
        'URL:"https://api.github.com/users/mendersoftware", '
        'EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/mendersoftware/followers", '
        'GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", '
        'ReposURL:"https://api.github.com/users/mendersoftware/repos", '
        'StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}, '
        'Name:"workflows", FullName:"mendersoftware/workflows", Description:"Workflow '
        'orchestrator for Mender", Homepage:"http://mender.io", '
        'DefaultBranch:"master", CreatedAt:github.Timestamp{2019-12-11 11:23:32 +0000 '
        "UTC}, PushedAt:github.Timestamp{2021-06-10 07:56:10 +0000 UTC}, UpdatedAt:github.Timestamp{2021-06-07 10:50:40 +0000 UTC}, "
        'HTMLURL:"https://github.com/mendersoftware/workflows", '
        'CloneURL:"https://github.com/mendersoftware/workflows.git", '
        'GitURL:"git://github.com/mendersoftware/workflows.git", '
        'SSHURL:"git@github.com:mendersoftware/workflows.git", '
        'SVNURL:"https://github.com/mendersoftware/workflows", Language:"Go", '
        "Fork:false, ForksCount:11, OpenIssuesCount:0, StargazersCount:3, WatchersCount:3, Size:5671, AllowRebaseMerge:true, AllowSquashMerge:true, AllowMergeCommit:true, Archived:false, Disabled:false, "
        'License:github.License{Key:"other", Name:"Other", SPDXID:"NOASSERTION"}, '
        "Private:false, HasIssues:true, HasWiki:true, HasPages:false, HasProjects:true, HasDownloads:true, "
        'URL:"https://api.github.com/repos/mendersoftware/workflows", '
        'ArchiveURL:"https://api.github.com/repos/mendersoftware/workflows/{archive_format}{/ref}", '
        'AssigneesURL:"https://api.github.com/repos/mendersoftware/workflows/assignees{/user}", '
        'BlobsURL:"https://api.github.com/repos/mendersoftware/workflows/git/blobs{/sha}", '
        'BranchesURL:"https://api.github.com/repos/mendersoftware/workflows/branches{/branch}", '
        'CollaboratorsURL:"https://api.github.com/repos/mendersoftware/workflows/collaborators{/collaborator}", '
        'CommentsURL:"https://api.github.com/repos/mendersoftware/workflows/comments{/number}", '
        'CommitsURL:"https://api.github.com/repos/mendersoftware/workflows/commits{/sha}", '
        'CompareURL:"https://api.github.com/repos/mendersoftware/workflows/compare/{base}...{head}", '
        'ContentsURL:"https://api.github.com/repos/mendersoftware/workflows/contents/{+path}", '
        'ContributorsURL:"https://api.github.com/repos/mendersoftware/workflows/contributors", '
        'DeploymentsURL:"https://api.github.com/repos/mendersoftware/workflows/deployments", '
        'DownloadsURL:"https://api.github.com/repos/mendersoftware/workflows/downloads", '
        'EventsURL:"https://api.github.com/repos/mendersoftware/workflows/events", '
        'ForksURL:"https://api.github.com/repos/mendersoftware/workflows/forks", '
        'GitCommitsURL:"https://api.github.com/repos/mendersoftware/workflows/git/commits{/sha}", '
        'GitRefsURL:"https://api.github.com/repos/mendersoftware/workflows/git/refs{/sha}", '
        'GitTagsURL:"https://api.github.com/repos/mendersoftware/workflows/git/tags{/sha}", '
        'HooksURL:"https://api.github.com/repos/mendersoftware/workflows/hooks", '
        'IssueCommentURL:"https://api.github.com/repos/mendersoftware/workflows/issues/comments{/number}", '
        'IssueEventsURL:"https://api.github.com/repos/mendersoftware/workflows/issues/events{/number}", '
        'IssuesURL:"https://api.github.com/repos/mendersoftware/workflows/issues{/number}", '
        'KeysURL:"https://api.github.com/repos/mendersoftware/workflows/keys{/key_id}", '
        'LabelsURL:"https://api.github.com/repos/mendersoftware/workflows/labels{/name}", '
        'LanguagesURL:"https://api.github.com/repos/mendersoftware/workflows/languages", '
        'MergesURL:"https://api.github.com/repos/mendersoftware/workflows/merges", '
        'MilestonesURL:"https://api.github.com/repos/mendersoftware/workflows/milestones{/number}", '
        'NotificationsURL:"https://api.github.com/repos/mendersoftware/workflows/notifications{?since,all,participating}", '
        'PullsURL:"https://api.github.com/repos/mendersoftware/workflows/pulls{/number}", '
        'ReleasesURL:"https://api.github.com/repos/mendersoftware/workflows/releases{/id}", '
        'StargazersURL:"https://api.github.com/repos/mendersoftware/workflows/stargazers", '
        'StatusesURL:"https://api.github.com/repos/mendersoftware/workflows/statuses/{sha}", '
        'SubscribersURL:"https://api.github.com/repos/mendersoftware/workflows/subscribers", '
        'SubscriptionURL:"https://api.github.com/repos/mendersoftware/workflows/subscription", '
        'TagsURL:"https://api.github.com/repos/mendersoftware/workflows/tags", '
        'TreesURL:"https://api.github.com/repos/mendersoftware/workflows/git/trees{/sha}", '
        'TeamsURL:"https://api.github.com/repos/mendersoftware/workflows/teams"}, '
        'User:github.User{Login:"mendersoftware", ID:15040539, '
        'NodeID:"MDEyOk9yZ2FuaXphdGlvbjE1MDQwNTM5", '
        'AvatarURL:"https://avatars.githubusercontent.com/u/15040539?v=4", '
        'HTMLURL:"https://github.com/mendersoftware", GravatarID:"", '
        'Type:"Organization", SiteAdmin:false, '
        'URL:"https://api.github.com/users/mendersoftware", '
        'EventsURL:"https://api.github.com/users/mendersoftware/events{/privacy}", '
        'FollowingURL:"https://api.github.com/users/mendersoftware/following{/other_user}", '
        'FollowersURL:"https://api.github.com/users/mendersoftware/followers", '
        'GistsURL:"https://api.github.com/users/mendersoftware/gists{/gist_id}", '
        'OrganizationsURL:"https://api.github.com/users/mendersoftware/orgs", '
        'ReceivedEventsURL:"https://api.github.com/users/mendersoftware/received_events", '
        'ReposURL:"https://api.github.com/users/mendersoftware/repos", '
        'StarredURL:"https://api.github.com/users/mendersoftware/starred{/owner}{/repo}", '
        'SubscriptionsURL:"https://api.github.com/users/mendersoftware/subscriptions"}}}',
        "debug:Trying to @mention the user in the newly created PR",
        "debug:userName: tranchitella",
        "github.CreateComment: "
        'org=mendersoftware,repo=workflows-enterprise,number=0,comment={"body":"@tranchitella '
        'I have created a PR for you, ready to merge as soon as tests are passed"}',
        "info:Pull request event with action: closed",
        "info:workflows:140 would trigger 0 builds",
        "debug:Got push event :: repo workflows-enterprise :: ref refs/heads/master",
        "git.Run: /usr/bin/git init .",
        "git.Run: /usr/bin/git remote add github git@github.com:/mendersoftware/workflows-enterprise.git",
        "git.Run: /usr/bin/git remote add gitlab git@gitlab.com:Northern.tech/Mender/workflows-enterprise",
        "git.Run: /usr/bin/git fetch github",
        "git.Run: /usr/bin/git checkout -b master github/master",
        "git.Run: /usr/bin/git push -f gitlab master",
        "info:Pushed ref to GitLab: workflows-enterprise:refs/heads/master",
        "info:Pull request event with action: opened",
        "git.Run: /usr/bin/git pull --rebase origin",
        "info:deviceconnect/master is being used in the following integration: [master]",
        "info:the following integration branches: [master] are using deviceconnect/master",
        "info:deviceconnect:109 will trigger 1 builds",
        "info:1: (main.buildOptions) {\n"
        ' pr: (string) (len=3) "109",\n'
        ' repo: (string) (len=13) "deviceconnect",\n'
        ' baseBranch: (string) (len=6) "master",\n'
        ' commitSHA: (string) (len=40) "c52542074ffe1c60dfceccf1baedf49dc10cb643",\n'
        " makeQEMU: (bool) false\n}\n\n",
        "info:auditlogs version origin/master is being used in master",
        "info:create-artifact-worker version origin/master is being used in master",
        "info:deployments version origin/master is being used in master",
        "info:deployments-enterprise version origin/master is being used in master",
        "info:deviceauth version origin/master is being used in master",
        "info:deviceconfig version origin/master is being used in master",
        "info:gui version origin/master is being used in master",
        "info:inventory version origin/master is being used in master",
        "info:inventory-enterprise version origin/master is being used in master",
        "info:mender version origin/master is being used in master",
        "info:mender-artifact version origin/master is being used in master",
        "info:mender-cli version origin/master is being used in master",
        "info:mender-connect version origin/master is being used in master",
        "info:mtls-ambassador version origin/master is being used in master",
        "info:tenantadm version origin/master is being used in master",
        "info:useradm version origin/master is being used in master",
        "info:useradm-enterprise version origin/master is being used in master",
        "info:workflows version origin/master is being used in master",
        "info:workflows-enterprise version origin/master is being used in master",
        "info:Creating pipeline in project Northern.tech/Mender/mender-qa:master with variables: AUDITLOGS_REV:origin/master, BUILD_BEAGLEBONEBLACK:, BUILD_CLIENT:false, BUILD_QEMUX86_64_BIOS_GRUB:, BUILD_QEMUX86_64_BIOS_GRUB_GPT:, BUILD_QEMUX86_64_UEFI_GRUB:, BUILD_VEXPRESS_QEMU:, BUILD_VEXPRESS_QEMU_FLASH:, BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, CREATE_ARTIFACT_WORKER_REV:origin/master, DEPLOYMENTS_ENTERPRISE_REV:origin/master, DEPLOYMENTS_REV:origin/master, DEVICEAUTH_REV:origin/master, DEVICECONFIG_REV:origin/master, DEVICECONNECT_REV:pull/109/head, GUI_REV:origin/master, INTEGRATION_REV:master, INVENTORY_ENTERPRISE_REV:origin/master, INVENTORY_REV:origin/master, MENDER_ARTIFACT_REV:origin/master, MENDER_CLI_REV:origin/master, MENDER_CONNECT_REV:origin/master, MENDER_REV:origin/master, MTLS_AMBASSADOR_REV:origin/master, RUN_INTEGRATION_TESTS:true, TENANTADM_REV:origin/master, TEST_QEMUX86_64_BIOS_GRUB:, TEST_QEMUX86_64_BIOS_GRUB_GPT:, TEST_QEMUX86_64_UEFI_GRUB:, TEST_VEXPRESS_QEMU:, TEST_VEXPRESS_QEMU_FLASH:, TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB:, USERADM_ENTERPRISE_REV:origin/master, USERADM_REV:origin/master, WORKFLOWS_ENTERPRISE_REV:origin/master, WORKFLOWS_REV:origin/master, ",
        "gitlab.CreatePipeline: "
        'path=Northern.tech/Mender/mender-qa,options={"ref":"master","variables":[{"key":"AUDITLOGS_REV","value":"origin/master"},{"key":"BUILD_BEAGLEBONEBLACK","value":""},{"key":"BUILD_CLIENT","value":"false"},{"key":"BUILD_QEMUX86_64_BIOS_GRUB","value":""},{"key":"BUILD_QEMUX86_64_BIOS_GRUB_GPT","value":""},{"key":"BUILD_QEMUX86_64_UEFI_GRUB","value":""},{"key":"BUILD_VEXPRESS_QEMU","value":""},{"key":"BUILD_VEXPRESS_QEMU_FLASH","value":""},{"key":"BUILD_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},{"key":"CREATE_ARTIFACT_WORKER_REV","value":"origin/master"},{"key":"DEPLOYMENTS_ENTERPRISE_REV","value":"origin/master"},{"key":"DEPLOYMENTS_REV","value":"origin/master"},{"key":"DEVICEAUTH_REV","value":"origin/master"},{"key":"DEVICECONFIG_REV","value":"origin/master"},{"key":"DEVICECONNECT_REV","value":"pull/109/head"},{"key":"GUI_REV","value":"origin/master"},{"key":"INTEGRATION_REV","value":"master"},{"key":"INVENTORY_ENTERPRISE_REV","value":"origin/master"},{"key":"INVENTORY_REV","value":"origin/master"},{"key":"MENDER_ARTIFACT_REV","value":"origin/master"},{"key":"MENDER_CLI_REV","value":"origin/master"},{"key":"MENDER_CONNECT_REV","value":"origin/master"},{"key":"MENDER_REV","value":"origin/master"},{"key":"MTLS_AMBASSADOR_REV","value":"origin/master"},{"key":"RUN_INTEGRATION_TESTS","value":"true"},{"key":"TENANTADM_REV","value":"origin/master"},{"key":"TEST_QEMUX86_64_BIOS_GRUB","value":""},{"key":"TEST_QEMUX86_64_BIOS_GRUB_GPT","value":""},{"key":"TEST_QEMUX86_64_UEFI_GRUB","value":""},{"key":"TEST_VEXPRESS_QEMU","value":""},{"key":"TEST_VEXPRESS_QEMU_FLASH","value":""},{"key":"TEST_VEXPRESS_QEMU_UBOOT_UEFI_GRUB","value":""},{"key":"USERADM_ENTERPRISE_REV","value":"origin/master"},{"key":"USERADM_REV","value":"origin/master"},{"key":"WORKFLOWS_ENTERPRISE_REV","value":"origin/master"},{"key":"WORKFLOWS_REV","value":"origin/master"}]}',
        "info:Created pipeline: ",
        "github.CreateComment: "
        'org=mendersoftware,repo=deviceconnect,number=109,comment={"body":"\\nHello '
        ":smile_cat: I created a pipeline for you here: [Pipeline-0]()\\n\\n\\u003cdetails\\u003e\\n    \\u003csummary\\u003eBuild Configuration Matrix\\u003c/summary\\u003e\\u003cp\\u003e\\n\\n| Key   | Value |\\n| ----- | ----- |\\n| AUDITLOGS_REV | origin/master |\\n| BUILD_CLIENT | false |\\n| CREATE_ARTIFACT_WORKER_REV | origin/master |\\n| DEPLOYMENTS_ENTERPRISE_REV | origin/master |\\n| DEPLOYMENTS_REV | origin/master |\\n| DEVICEAUTH_REV | origin/master |\\n| DEVICECONFIG_REV | origin/master |\\n| DEVICECONNECT_REV | pull/109/head |\\n| GUI_REV | origin/master |\\n| INTEGRATION_REV | master |\\n| INVENTORY_ENTERPRISE_REV | origin/master |\\n| INVENTORY_REV | origin/master |\\n| MENDER_ARTIFACT_REV | origin/master |\\n| MENDER_CLI_REV | origin/master |\\n| MENDER_CONNECT_REV | origin/master |\\n| MENDER_REV | origin/master |\\n| MTLS_AMBASSADOR_REV | origin/master |\\n| RUN_INTEGRATION_TESTS | true |\\n| TENANTADM_REV | origin/master |\\n| USERADM_ENTERPRISE_REV | origin/master |\\n| USERADM_REV | origin/master |\\n| WORKFLOWS_ENTERPRISE_REV | origin/master |\\n| WORKFLOWS_REV | origin/master "
        '|\\n\\n\\n \\u003c/p\\u003e\\u003c/details\\u003e\\n"}',
    ]
