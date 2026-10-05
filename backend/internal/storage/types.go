package storage

import (
	"context"

	"snorlx/backend/internal/models"
)

// Storage defines the interface for all storage operations.
// This allows swapping between in-memory and database storage.
//
// Methods that take a userID return only data the user may see: a repository is visible to a user
// when GrantRepositoryAccess was recorded for that pair (the user synced it with their own GitHub
// token). Single-object getters without userID are for internal use; HTTP handlers must check
// HasRepositoryAccess before returning them.
type Storage interface {
	// Lifecycle
	Close() error
	Migrate() error
	// Ping reports whether the backing store is reachable (readiness).
	Ping(ctx context.Context) error

	// Organizations
	ListOrganizations(ctx context.Context, userID int) ([]models.Organization, error)
	GetOrganization(ctx context.Context, userID, id int) (*models.Organization, error)
	GetOrganizationByGitHubID(ctx context.Context, githubID int64) (*models.Organization, error)
	UpsertOrganization(ctx context.Context, org *models.Organization) (*models.Organization, error)

	// Repositories
	ListRepositories(ctx context.Context, userID, page, pageSize int, search string) ([]models.Repository, int, error)
	GetRepository(ctx context.Context, id int) (*models.Repository, error)
	GetRepositoryByGitHubID(ctx context.Context, githubID int64) (*models.Repository, error)
	UpsertRepository(ctx context.Context, repo *models.Repository) (*models.Repository, error)
	UpdateRepository(ctx context.Context, id int, repo *models.Repository) (*models.Repository, error)

	// Repository access (tenancy)
	GrantRepositoryAccess(ctx context.Context, userID, repoID int) error
	HasRepositoryAccess(ctx context.Context, userID, repoID int) (bool, error)
	ListUsersWithRepositoryAccess(ctx context.Context, repoID int) ([]int, error)

	// Workflows
	ListWorkflows(ctx context.Context, userID int, repoID *int) ([]models.Workflow, error)
	GetWorkflow(ctx context.Context, id int) (*models.Workflow, error)
	GetWorkflowByGitHubID(ctx context.Context, githubID int64) (*models.Workflow, error)
	UpsertWorkflow(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error)
	UpdateWorkflow(ctx context.Context, id int, workflow *models.Workflow) (*models.Workflow, error)

	// Workflow Runs
	ListRuns(ctx context.Context, userID int, filters *models.RunFilters, page, pageSize int) ([]models.WorkflowRun, int, error)
	ListActivePipelines(ctx context.Context, userID int) ([]models.WorkflowRun, error)
	GetRun(ctx context.Context, id int) (*models.WorkflowRun, error)
	GetRunByGitHubID(ctx context.Context, githubID int64) (*models.WorkflowRun, error)
	UpsertRun(ctx context.Context, run *models.WorkflowRun) (*models.WorkflowRun, error)

	// Workflow Jobs
	ListJobsForRun(ctx context.Context, runID int) ([]models.WorkflowJob, error)
	GetJob(ctx context.Context, id int) (*models.WorkflowJob, error)
	UpsertJob(ctx context.Context, job *models.WorkflowJob) (*models.WorkflowJob, error)

	// Deployments
	ListDeployments(ctx context.Context, repoID *int) ([]models.Deployment, error)
	GetDeployment(ctx context.Context, id int) (*models.Deployment, error)
	UpsertDeployment(ctx context.Context, deployment *models.Deployment) (*models.Deployment, error)

	// Users & Sessions
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	GetUserByGitHubID(ctx context.Context, githubID int64) (*models.User, error)
	UpsertUser(ctx context.Context, user *models.User) (*models.User, error)
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, sessionID string) (*models.Session, *models.User, error)
	DeleteSession(ctx context.Context, sessionID string) error
	CleanExpiredSessions(ctx context.Context) error

	// API tokens (MCP / personal access)
	CreateApiToken(ctx context.Context, token *models.ApiToken) (*models.ApiToken, error)
	ListApiTokens(ctx context.Context, userID int) ([]models.ApiToken, error)
	GetApiTokenByHash(ctx context.Context, tokenHash string) (*models.ApiToken, *models.User, error)
	RevokeApiToken(ctx context.Context, userID, tokenID int) error
	TouchApiTokenLastUsed(ctx context.Context, tokenID int) error

	// Dashboard
	GetDashboardSummary(ctx context.Context, userID int) (*models.DashboardSummary, error)
	GetTrends(ctx context.Context, userID, days int) ([]models.Trend, error)

	// Backfill
	BackfillDeploymentRuns(ctx context.Context, userID int) (int, error)

	// Repository Scores
	UpsertRepositoryScore(ctx context.Context, score *models.RepositoryScore) (*models.RepositoryScore, error)
	GetLatestRepositoryScore(ctx context.Context, repoID int) (*models.RepositoryScore, error)
	ListLatestRepositoryScores(ctx context.Context, userID int) ([]models.RepositoryScore, error)
}

// StorageMode defines the storage backend type
type StorageMode string

const (
	StorageModeMemory   StorageMode = "memory"
	StorageModeDatabase StorageMode = "database"
)
