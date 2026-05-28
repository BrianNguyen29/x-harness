package classify

import (
	"slices"
	"testing"
)

func TestClassifyCommandGitStatus(t *testing.T) {
	c := ClassifyCommand("git status --porcelain")
	if !slices.Contains(c.Intents, "read_files") {
		t.Fatalf("expected read_files intent, got %v", c.Intents)
	}
	if c.Risk != "low" {
		t.Fatalf("expected low risk, got %s", c.Risk)
	}
	if c.Unknown {
		t.Fatal("expected unknown=false")
	}
}

func TestClassifyCommandGitDiff(t *testing.T) {
	c := ClassifyCommand("git diff --name-only")
	if !slices.Contains(c.Intents, "read_files") {
		t.Fatalf("expected read_files intent, got %v", c.Intents)
	}
	if c.Risk != "low" {
		t.Fatalf("expected low risk, got %s", c.Risk)
	}
}

func TestClassifyCommandGoTest(t *testing.T) {
	c := ClassifyCommand("go test ./...")
	if !slices.Contains(c.Intents, "read_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected read_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "low" {
		t.Fatalf("expected low risk, got %s", c.Risk)
	}
}

func TestClassifyCommandNPMTest(t *testing.T) {
	c := ClassifyCommand("npm test")
	if !slices.Contains(c.Intents, "read_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected read_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "low" {
		t.Fatalf("expected low risk, got %s", c.Risk)
	}
}

func TestClassifyCommandGoBuild(t *testing.T) {
	c := ClassifyCommand("go build ./cmd/x-harness")
	if !slices.Contains(c.Intents, "write_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected write_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "medium" {
		t.Fatalf("expected medium risk, got %s", c.Risk)
	}
}

func TestClassifyCommandNPMBuild(t *testing.T) {
	c := ClassifyCommand("npm run build")
	if !slices.Contains(c.Intents, "write_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected write_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "medium" {
		t.Fatalf("expected medium risk, got %s", c.Risk)
	}
}

func TestClassifyCommandNPMInstall(t *testing.T) {
	c := ClassifyCommand("npm install")
	if !slices.Contains(c.Intents, "package_install") || !slices.Contains(c.Intents, "network_outbound") {
		t.Fatalf("expected package_install and network_outbound intents, got %v", c.Intents)
	}
	if c.Risk != "medium" {
		t.Fatalf("expected medium risk, got %s", c.Risk)
	}
}

func TestClassifyCommandNPMPublish(t *testing.T) {
	c := ClassifyCommand("npm publish")
	if !slices.Contains(c.Intents, "package_publish") || !slices.Contains(c.Intents, "network_outbound") {
		t.Fatalf("expected package_publish and network_outbound intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandRMRF(t *testing.T) {
	c := ClassifyCommand("rm -rf /some/path")
	if !slices.Contains(c.Intents, "delete_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected delete_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandCurl(t *testing.T) {
	c := ClassifyCommand("curl https://example.com")
	if !slices.Contains(c.Intents, "network_outbound") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected network_outbound and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandWget(t *testing.T) {
	c := ClassifyCommand("wget https://example.com/file.tar.gz")
	if !slices.Contains(c.Intents, "network_outbound") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected network_outbound and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandGitPush(t *testing.T) {
	c := ClassifyCommand("git push origin main")
	if !slices.Contains(c.Intents, "git_mutation") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected git_mutation and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandSedInPlace(t *testing.T) {
	c := ClassifyCommand("sed -i 's/old/new/g' file.txt")
	if !slices.Contains(c.Intents, "write_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected write_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "medium" {
		t.Fatalf("expected medium risk, got %s", c.Risk)
	}
}

func TestClassifyCommandAWS(t *testing.T) {
	c := ClassifyCommand("aws s3 ls")
	if !slices.Contains(c.Intents, "secret_access") || !slices.Contains(c.Intents, "permission_change") {
		t.Fatalf("expected secret_access and permission_change intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandGCloud(t *testing.T) {
	c := ClassifyCommand("gcloud compute instances list")
	if !slices.Contains(c.Intents, "secret_access") || !slices.Contains(c.Intents, "permission_change") {
		t.Fatalf("expected secret_access and permission_change intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandAZ(t *testing.T) {
	c := ClassifyCommand("az group list")
	if !slices.Contains(c.Intents, "secret_access") || !slices.Contains(c.Intents, "permission_change") {
		t.Fatalf("expected secret_access and permission_change intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandPSQL(t *testing.T) {
	c := ClassifyCommand("psql -d mydb -c 'SELECT 1'")
	if !slices.Contains(c.Intents, "database_mutation") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected database_mutation and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandMySQL(t *testing.T) {
	c := ClassifyCommand("mysql -u root -e 'SHOW DATABASES'")
	if !slices.Contains(c.Intents, "database_mutation") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected database_mutation and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandSQLite(t *testing.T) {
	c := ClassifyCommand("sqlite3 data.db '.tables'")
	if !slices.Contains(c.Intents, "database_mutation") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected database_mutation and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandUnknown(t *testing.T) {
	c := ClassifyCommand("custom-tool --flag")
	if !slices.Contains(c.Intents, "unknown") {
		t.Fatalf("expected unknown intent, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk for unknown, got %s", c.Risk)
	}
	if !c.Unknown {
		t.Fatal("expected unknown=true")
	}
}

func TestClassifyCommandEmpty(t *testing.T) {
	c := ClassifyCommand("")
	if !slices.Contains(c.Intents, "unknown") {
		t.Fatalf("expected unknown intent, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk for empty command, got %s", c.Risk)
	}
	if !c.Unknown {
		t.Fatal("expected unknown=true")
	}
}

func TestClassifyCommandKubectlApply(t *testing.T) {
	c := ClassifyCommand("kubectl apply -f deployment.yaml")
	if !slices.Contains(c.Intents, "deploy_or_publish") || !slices.Contains(c.Intents, "network_outbound") {
		t.Fatalf("expected deploy_or_publish and network_outbound intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandTerraformApply(t *testing.T) {
	c := ClassifyCommand("terraform apply -auto-approve")
	if !slices.Contains(c.Intents, "deploy_or_publish") || !slices.Contains(c.Intents, "network_outbound") {
		t.Fatalf("expected deploy_or_publish and network_outbound intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandChmod(t *testing.T) {
	c := ClassifyCommand("chmod +x script.sh")
	if !slices.Contains(c.Intents, "permission_change") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected permission_change and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandCat(t *testing.T) {
	c := ClassifyCommand("cat file.txt")
	if !slices.Contains(c.Intents, "read_files") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected read_files and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "low" {
		t.Fatalf("expected low risk, got %s", c.Risk)
	}
}

func TestClassifyCommandMultipleIntents(t *testing.T) {
	// npm publish should have package_publish, network_outbound, shell_exec
	c := ClassifyCommand("npm publish --tag beta")
	expected := []string{"package_publish", "network_outbound", "shell_exec"}
	for _, exp := range expected {
		if !slices.Contains(c.Intents, exp) {
			t.Fatalf("expected intent %s, got %v", exp, c.Intents)
		}
	}
}

func TestClassifyCommandServerlessDeploy(t *testing.T) {
	c := ClassifyCommand("serverless deploy --stage prod")
	if !slices.Contains(c.Intents, "deploy_or_publish") || !slices.Contains(c.Intents, "network_outbound") {
		t.Fatalf("expected deploy_or_publish and network_outbound intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}

func TestClassifyCommandSudo(t *testing.T) {
	c := ClassifyCommand("sudo apt-get update")
	if !slices.Contains(c.Intents, "permission_change") || !slices.Contains(c.Intents, "shell_exec") {
		t.Fatalf("expected permission_change and shell_exec intents, got %v", c.Intents)
	}
	if c.Risk != "high" {
		t.Fatalf("expected high risk, got %s", c.Risk)
	}
}
