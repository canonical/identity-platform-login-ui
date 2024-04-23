local claims = std.extVar('claims');

{
  identity: {
    traits: {
      [if 'email' in claims then 'email' else null]: claims.email,
      [if 'name' in claims then 'name' else null]: claims.name,
      [if 'given_name' in claims then 'given_name' else null]: claims.given_name,
      [if 'family_name' in claims then 'family_name' else null]: claims.family_name,
      [if 'last_name' in claims then 'last_name' else null]: claims.last_name,
      [if 'middle_name' in claims then 'middle_name' else null]: claims.middle_name,
      [if 'nickname' in claims then 'nickname' else null]: claims.nickname,
      [if 'preferred_username' in claims then 'preferred_username' else null]: claims.preferred_username,
      [if 'profile' in claims then 'profile' else null]: claims.profile,
      [if 'picture' in claims then 'picture' else null]: claims.picture,
      [if 'website' in claims then 'website' else null]: claims.website,
      [if 'gender' in claims then 'gender' else null]: claims.gender,
      [if 'birthdate' in claims then 'birthdate' else null]: claims.birthdate,
      [if 'zoneinfo' in claims then 'zoneinfo' else null]: claims.zoneinfo,
      [if 'phone_number' in claims && claims.phone_number_verified then 'phone_number' else null]: claims.phone_number,
      [if 'locale' in claims then 'locale' else null]: claims.locale,
      [if 'team' in claims then 'team' else null]: claims.team,
    },
  },
}
