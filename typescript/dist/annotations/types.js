"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AuthType = exports.DeploymentType = void 0;
/**
 * Deployment types for Box handlers
 */
var DeploymentType;
(function (DeploymentType) {
    DeploymentType["Function"] = "function";
    DeploymentType["Container"] = "container";
})(DeploymentType || (exports.DeploymentType = DeploymentType = {}));
/**
 * Authentication requirement levels
 */
var AuthType;
(function (AuthType) {
    AuthType["None"] = "none";
    AuthType["Optional"] = "optional";
    AuthType["Required"] = "required";
})(AuthType || (exports.AuthType = AuthType = {}));
//# sourceMappingURL=types.js.map