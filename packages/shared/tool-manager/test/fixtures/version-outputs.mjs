/**
 * Sample version command outputs for testing.
 */
export const VERSION_OUTPUTS = {
  git: {
    stdout: 'git version 2.41.0\n',
    stderr: '',
    exitCode: 0,
  },
  gitOld: {
    stdout: 'git version 2.20.0\n',
    stderr: '',
    exitCode: 0,
  },
  aws: {
    stdout: 'aws-cli/2.15.0 Python/3.11.6 Darwin/23.0.0 source/arm64 prompt/off\n',
    stderr: '',
    exitCode: 0,
  },
  gh: {
    stdout: 'gh version 2.32.0 (2023-07-14)\nhttps://github.com/cli/cli/releases/tag/v2.32.0\n',
    stderr: '',
    exitCode: 0,
  },
  maven: {
    stdout: 'Apache Maven 3.9.4 (dfbb324ad4a7c8fb0bf182e6d91b0ae20e3d2dd9)\nMaven home: /usr/local/Cellar/maven/3.9.4/libexec\n',
    stderr: '',
    exitCode: 0,
  },
  kubectl: {
    stdout: 'Client Version: v1.28.2\nKustomize Version: v5.0.4-0.20230601165947-6ce0bf390ce3\n',
    stderr: '',
    exitCode: 0,
  },
  docker: {
    stdout: 'Docker version 24.0.6, build ed223bc\n',
    stderr: '',
    exitCode: 0,
  },
  notFound: {
    stdout: '',
    stderr: '',
    exitCode: 127,
  },
  whichFound: {
    stdout: '/usr/local/bin/brew\n',
    stderr: '',
    exitCode: 0,
  },
  whichNotFound: {
    stdout: '',
    stderr: '',
    exitCode: 1,
  },
  installSuccess: {
    stdout: 'Successfully installed.\n',
    stderr: '',
    exitCode: 0,
  },
  installFailure: {
    stdout: '',
    stderr: 'Error: installation failed\n',
    exitCode: 1,
  },
};
