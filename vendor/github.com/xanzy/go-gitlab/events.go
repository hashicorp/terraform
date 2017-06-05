//
// Copyright 2015, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import "time"

// PushEvent represents a push event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#push-events
type PushEvent struct {
	ObjectKind  string `json:"object_kind"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Ref         string `json:"ref"`
	CheckoutSha string `json:"checkout_sha"`
	UserID      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	UserAvatar  string `json:"user_avatar"`
	ProjectID   int    `json:"project_id"`
	Project     struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository        *Repository `json:"repository"`
	Commits           []*Commit   `json:"commits"`
	TotalCommitsCount int         `json:"total_commits_count"`
}

// TagEvent represents a tag event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#tag-events
type TagEvent struct {
	ObjectKind  string `json:"object_kind"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Ref         string `json:"ref"`
	CheckoutSha string `json:"checkout_sha"`
	UserID      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	UserAvatar  string `json:"user_avatar"`
	ProjectID   int    `json:"project_id"`
	Project     struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository        *Repository `json:"repository"`
	Commits           []*Commit   `json:"commits"`
	TotalCommitsCount int         `json:"total_commits_count"`
}

// IssueEvent represents a issue event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#issues-events
type IssueEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository       *Repository `json:"repository"`
	ObjectAttributes struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		AssigneeID  int    `json:"assignee_id"`
		AuthorID    int    `json:"author_id"`
		ProjectID   int    `json:"project_id"`
		CreatedAt   string `json:"created_at"` // Should be *time.Time (see Gitlab issue #21468)
		UpdatedAt   string `json:"updated_at"` // Should be *time.Time (see Gitlab issue #21468)
		Position    int    `json:"position"`
		BranchName  string `json:"branch_name"`
		Description string `json:"description"`
		MilestoneID int    `json:"milestone_id"`
		State       string `json:"state"`
		Iid         int    `json:"iid"`
		URL         string `json:"url"`
		Action      string `json:"action"`
	} `json:"object_attributes"`
	Assignee struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"assignee"`
}

// CommitCommentEvent represents a comment on a commit event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#comment-on-commit
type CommitCommentEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	ProjectID  int    `json:"project_id"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository       *Repository `json:"repository"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		AuthorID     int    `json:"author_id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		ProjectID    int    `json:"project_id"`
		Attachment   string `json:"attachment"`
		LineCode     string `json:"line_code"`
		CommitID     string `json:"commit_id"`
		NoteableID   int    `json:"noteable_id"`
		System       bool   `json:"system"`
		StDiff       struct {
			Diff        string `json:"diff"`
			NewPath     string `json:"new_path"`
			OldPath     string `json:"old_path"`
			AMode       string `json:"a_mode"`
			BMode       string `json:"b_mode"`
			NewFile     bool   `json:"new_file"`
			RenamedFile bool   `json:"renamed_file"`
			DeletedFile bool   `json:"deleted_file"`
		} `json:"st_diff"`
	} `json:"object_attributes"`
	Commit *Commit `json:"commit"`
}

// MergeCommentEvent represents a comment on a merge event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#comment-on-merge-request
type MergeCommentEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	ProjectID  int    `json:"project_id"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository       *Repository `json:"repository"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		AuthorID     int    `json:"author_id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		ProjectID    int    `json:"project_id"`
		Attachment   string `json:"attachment"`
		LineCode     string `json:"line_code"`
		CommitID     string `json:"commit_id"`
		NoteableID   int    `json:"noteable_id"`
		System       bool   `json:"system"`
		StDiff       *Diff  `json:"st_diff"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
	MergeRequest *MergeRequest `json:"merge_request"`
}

// IssueCommentEvent represents a comment on an issue event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#comment-on-issue
type IssueCommentEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	ProjectID  int    `json:"project_id"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository       *Repository `json:"repository"`
	ObjectAttributes struct {
		ID           int     `json:"id"`
		Note         string  `json:"note"`
		NoteableType string  `json:"noteable_type"`
		AuthorID     int     `json:"author_id"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
		ProjectID    int     `json:"project_id"`
		Attachment   string  `json:"attachment"`
		LineCode     string  `json:"line_code"`
		CommitID     string  `json:"commit_id"`
		NoteableID   int     `json:"noteable_id"`
		System       bool    `json:"system"`
		StDiff       []*Diff `json:"st_diff"`
		URL          string  `json:"url"`
	} `json:"object_attributes"`
	Issue *Issue `json:"issue"`
}

// SnippetCommentEvent represents a comment on a snippet event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#comment-on-code-snippet
type SnippetCommentEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	ProjectID  int    `json:"project_id"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Repository       *Repository `json:"repository"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		AuthorID     int    `json:"author_id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		ProjectID    int    `json:"project_id"`
		Attachment   string `json:"attachment"`
		LineCode     string `json:"line_code"`
		CommitID     string `json:"commit_id"`
		NoteableID   int    `json:"noteable_id"`
		System       bool   `json:"system"`
		StDiff       *Diff  `json:"st_diff"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
	Snippet *Snippet `json:"snippet"`
}

// MergeEvent represents a merge event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#merge-request-events
type MergeEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	ObjectAttributes struct {
		ID              int       `json:"id"`
		TargetBranch    string    `json:"target_branch"`
		SourceBranch    string    `json:"source_branch"`
		SourceProjectID int       `json:"source_project_id"`
		AuthorID        int       `json:"author_id"`
		AssigneeID      int       `json:"assignee_id"`
		Title           string    `json:"title"`
		CreatedAt       string    `json:"created_at"` // Should be *time.Time (see Gitlab issue #21468)
		UpdatedAt       string    `json:"updated_at"` // Should be *time.Time (see Gitlab issue #21468)
		StCommits       []*Commit `json:"st_commits"`
		StDiffs         []*Diff   `json:"st_diffs"`
		MilestoneID     int       `json:"milestone_id"`
		State           string    `json:"state"`
		MergeStatus     string    `json:"merge_status"`
		TargetProjectID int       `json:"target_project_id"`
		Iid             int       `json:"iid"`
		Description     string    `json:"description"`
		Position        int       `json:"position"`
		LockedAt        string    `json:"locked_at"`
		UpdatedByID     int       `json:"updated_by_id"`
		MergeError      string    `json:"merge_error"`
		MergeParams     struct {
			ForceRemoveSourceBranch string `json:"force_remove_source_branch"`
		} `json:"merge_params"`
		MergeWhenBuildSucceeds   bool        `json:"merge_when_build_succeeds"`
		MergeUserID              int         `json:"merge_user_id"`
		MergeCommitSha           string      `json:"merge_commit_sha"`
		DeletedAt                string      `json:"deleted_at"`
		ApprovalsBeforeMerge     string      `json:"approvals_before_merge"`
		RebaseCommitSha          string      `json:"rebase_commit_sha"`
		InProgressMergeCommitSha string      `json:"in_progress_merge_commit_sha"`
		LockVersion              int         `json:"lock_version"`
		TimeEstimate             int         `json:"time_estimate"`
		Source                   *Repository `json:"source"`
		Target                   *Repository `json:"target"`
		LastCommit               struct {
			ID        string     `json:"id"`
			Message   string     `json:"message"`
			Timestamp *time.Time `json:"timestamp"`
			URL       string     `json:"url"`
			Author    *Author    `json:"author"`
		} `json:"last_commit"`
		WorkInProgress bool   `json:"work_in_progress"`
		URL            string `json:"url"`
		Action         string `json:"action"`
		Assignee       struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"assignee"`
	} `json:"object_attributes"`
	Repository *Repository `json:"repository"`
	Assignee   struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"assignee"`
}

// WikiPageEvent represents a wiki page event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#wiki-page-events
type WikiPageEvent struct {
	ObjectKind string `json:"object_kind"`
	User       *User  `json:"user"`
	Project    struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Wiki struct {
		WebURL            string `json:"web_url"`
		GitSSHURL         string `json:"git_ssh_url"`
		GitHTTPURL        string `json:"git_http_url"`
		PathWithNamespace string `json:"path_with_namespace"`
		DefaultBranch     string `json:"default_branch"`
	} `json:"wiki"`
	ObjectAttributes struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Format  string `json:"format"`
		Message string `json:"message"`
		Slug    string `json:"slug"`
		URL     string `json:"url"`
		Action  string `json:"action"`
	} `json:"object_attributes"`
}

// PipelineEvent represents a pipeline event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#pipeline-events
type PipelineEvent struct {
	ObjectKind       string `json:"object_kind"`
	ObjectAttributes struct {
		ID         int      `json:"id"`
		Ref        string   `json:"ref"`
		Tag        bool     `json:"tag"`
		Sha        string   `json:"sha"`
		BeforeSha  string   `json:"before_sha"`
		Status     string   `json:"status"`
		Stages     []string `json:"stages"`
		CreatedAt  string   `json:"created_at"`
		FinishedAt string   `json:"finished_at"`
		Duration   int      `json:"duration"`
	} `json:"object_attributes"`
	User struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Project struct {
		Name              string               `json:"name"`
		Description       string               `json:"description"`
		AvatarURL         string               `json:"avatar_url"`
		GitSSHURL         string               `json:"git_ssh_url"`
		GitHTTPURL        string               `json:"git_http_url"`
		Namespace         string               `json:"namespace"`
		PathWithNamespace string               `json:"path_with_namespace"`
		DefaultBranch     string               `json:"default_branch"`
		Homepage          string               `json:"homepage"`
		URL               string               `json:"url"`
		SSHURL            string               `json:"ssh_url"`
		HTTPURL           string               `json:"http_url"`
		WebURL            string               `json:"web_url"`
		VisibilityLevel   VisibilityLevelValue `json:"visibility_level"`
	} `json:"project"`
	Commit struct {
		ID        string    `json:"id"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
		URL       string    `json:"url"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commit"`
	Builds []struct {
		ID         int    `json:"id"`
		Stage      string `json:"stage"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
		When       string `json:"when"`
		Manual     bool   `json:"manual"`
		User       struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		Runner        string `json:"runner"`
		ArtifactsFile struct {
			Filename string `json:"filename"`
			Size     string `json:"size"`
		} `json:"artifacts_file"`
	} `json:"builds"`
}

//BuildEvent represents a build event
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/web_hooks/web_hooks.md#build-events
type BuildEvent struct {
	ObjectKind        string `json:"object_kind"`
	Ref               string `json:"ref"`
	Tag               bool   `json:"tag"`
	BeforeSha         string `json:"before_sha"`
	Sha               string `json:"sha"`
	BuildID           int    `json:"build_id"`
	BuildName         string `json:"build_name"`
	BuildStage        string `json:"build_stage"`
	BuildStatus       string `json:"build_status"`
	BuildStartedAt    string `json:"build_started_at"`
	BuildFinishedAt   string `json:"build_finished_at"`
	BuildDuration     string `json:"build_duration"`
	BuildAllowFailure bool   `json:"build_allow_failure"`
	ProjectID         int    `json:"project_id"`
	ProjectName       string `json:"project_name"`
	User              struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
	Commit struct {
		ID          int    `json:"id"`
		Sha         string `json:"sha"`
		Message     string `json:"message"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
		Status      string `json:"status"`
		Duration    string `json:"duration"`
		StartedAt   string `json:"started_at"`
		FinishedAt  string `json:"finished_at"`
	} `json:"commit"`
	Repository *Repository `json:"repository"`
}
