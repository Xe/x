/* Import dependencies, declare constants */

win = (callback) => {
    callback(null, "allowed");
};

fail = (callback) => {
    callback("not allowed");
};

/**
* Your function call
* @param {Object} params Execution parameters
*   Members
*   - {Array} args Arguments passed to function
*   - {Object} kwargs Keyword arguments (key-value pairs) passed to function
*   - {String} remoteAddress The IPv4 or IPv6 address of the caller
*
* @param {Function} callback Execute this to end the function call
*   Arguments
*   - {Error} error The error to show if function fails
*   - {Any} returnValue JSON serializable (or Buffer) return value
*/
module.exports = (params, callback) => {
    switch (params.kwargs.user) {
    case "Xena":
        win(callback);

    default:
        fail(callback);
    }
};
