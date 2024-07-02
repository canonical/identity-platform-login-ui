export const isValidUrl = (input: string) => {
  try {
    new URL(input);
    return true;
  } catch (e) {
    return false;
  }
};
