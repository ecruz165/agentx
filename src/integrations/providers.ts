export interface ProviderConfig {
  configDir: string;
  mainDoc: {
    template: string;
    filename: string;
    atProjectRoot: boolean;
  };
  commands: {
    supported: boolean;
    template?: string;
  };
  context: {
    subdir: string;
  };
  renders: {
    skills: boolean;
    workflows: boolean;
  };
}

export const PROVIDERS: Record<string, ProviderConfig> = {
  'claude-code': {
    configDir: '.claude',
    mainDoc: {
      template: 'main-doc.hbs',
      filename: 'CLAUDE.md',
      atProjectRoot: false,
    },
    commands: {
      supported: true,
      template: 'command.hbs',
    },
    context: {
      subdir: 'context',
    },
    renders: {
      skills: true,
      workflows: true,
    },
  },
  augment: {
    configDir: '.augment',
    mainDoc: {
      template: 'main-doc.hbs',
      filename: 'augment-guidelines.md',
      atProjectRoot: false,
    },
    commands: {
      supported: false,
    },
    context: {
      subdir: 'context',
    },
    renders: {
      skills: false,
      workflows: false,
    },
  },
  opencode: {
    configDir: '.opencode',
    mainDoc: {
      template: 'main-doc.hbs',
      filename: 'AGENTS.md',
      atProjectRoot: true,
    },
    commands: {
      supported: true,
      template: 'command.hbs',
    },
    context: {
      subdir: 'context',
    },
    renders: {
      skills: true,
      workflows: true,
    },
  },
  copilot: {
    configDir: '.github',
    mainDoc: {
      template: 'main-doc.hbs',
      filename: 'copilot-instructions.md',
      atProjectRoot: false,
    },
    commands: {
      supported: false,
    },
    context: {
      subdir: 'copilot-context',
    },
    renders: {
      skills: false,
      workflows: false,
    },
  },
};
