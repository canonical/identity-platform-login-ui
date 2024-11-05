import { execSync } from "child_process";

export const resetIdentities = () => {
  deleteIdentity();
  createIdentity();
};

export const deleteIdentity = () => {
  execSync(
    "kratos delete identity  --endpoint http://localhost:4434 $(kratos list identities --endpoint http://localhost:4434 --format json | jq -r .identities[].id)",
  );
};

export const createIdentity = () => {
  execSync(
    "kratos import identities ../docker/kratos/identity.json --endpoint http://localhost:4434",
  );
};
