import { execSync } from "child_process";

export const resetIdentities = () => {
  deleteIdentity();
  createIdentity();
};

export const deleteIdentity = () => {
  execSync(
    'ID=$(curl --silent -H "Content-Type: application/json" -X GET "http://localhost:4434/admin/identities" | jq -r .[0].id) && curl --silent -H "Accept: application/json" -X DELETE "http://localhost:4434/admin/identities/$ID"',
  );
};

export const createIdentity = () => {
  execSync(
    'curl --silent -H "Content-Type: application/json" -X POST "http://localhost:4434/admin/identities" -d @../docker/kratos/identity.json',
  );
};
