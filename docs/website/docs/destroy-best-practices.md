---
layout: "docs"
page_title: "Destroy Best Practices"
description: |-
  Recommended practices for safely using `terraform destroy` to avoid accidental data loss.
---

# Destroy Best Practices

The `terraform destroy` command removes all resources defined in your configuration.  
While this can be useful for cleaning up infrastructure, it can also lead to accidental data loss if not used carefully.  
Follow these best practices to ensure safe and controlled destruction of infrastructure:

---

## 1. Use `-target` Carefully
If you only want to destroy specific resources, you can use the `-target` flag:  

```bash
terraform destroy -target=aws_instance.example
```


## 2. Always Review the Plan First

Before running a full destroy, generate and inspect the plan:

```bash
terraform plan -destroy
```


This shows exactly what resources will be removed, helping you avoid surprises.

## 3. Protect Critical Resources

For resources that should never be destroyed (e.g., databases, production load balancers), use the `prevent_destroy` lifecycle rule in your `.tf` configuration:

```hcl
resource "aws_db_instance" "prod" {
  # ...
  lifecycle {
    prevent_destroy = true
  }
}
```


This ensures Terraform fails if you try to destroy these resources.

## 4. Back Up state files

Always back up your .tfstate files before running `terraform destroy`.
You can use remote backends like S3 with DynamoDB locking to protect state consistency.

## 5. Use Workspaces for Isolation

Separate environments (e.g., dev, staging, prod) with workspaces:

```bash
terraform workspace new dev
terraform workspace select dev
```


This prevents accidental destruction of production infrastructure when testing.

## 6. Confirm with Your Team

If you work in a shared environment, never run terraform destroy without alignment.
Coordinate with your team to avoid unexpected downtime.

## Conclusion

The terraform destroy command is powerful, but with great power comes great responsibility.
By following these best practices, you can ensure that destruction is intentional, controlled, and safe.


---

