/**
 * Takes an array of strings and returns
 * the longest common substring they share
 *
 * @param {Array} strings Strings to check
 * @returns {String} the longest common substring
 */
function findLCS(strings) {
  // Sort the strings in alphabetical order, in a new shallow copy
  const sortedArray = strings.concat().sort()

  // Get the first and last strings
  // This will set a limit on how many characters to compare
  const firstString = sortedArray[0]
  const lastString = sortedArray[sortedArray.length - 1]

  /**
   * keep checking characters to see
   * if they're the same, as long as:
   * - there's still characters to compare
   *   from the first string
   * - The current check returns true
   */
  let i = 0
  while (
    i < firstString.length &&
    firstString.charAt(i) === lastString.charAt(i)
  )
    i++

  /**
   * The longest comment substring ends
   * where the loop does, so return that here
   */
  return firstString.substring(0, i)
}

module.exports = findLCS