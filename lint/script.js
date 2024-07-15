core.debug(`eventName: ${context.eventName}`);
if (context.eventName != 'pull_request') {
  core.info('skipping event');
  return;
}

const call = github.paginate(
  github.rest.pulls.listCommits,
  {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.payload.pull_request,
  },
  (res) =>
    res.data.map((obj) => { obj.sha, obj.commit.message }),
);

const regexp = new Regexp(core.getInput('pattern'), core.getInput('flags'));
const failed = [];

for await (const commit of github.paginate(github.rest.pulls.listCommits, args, mapcommit)) {
  regexp.lastIndex = 0;
  const ok = regexp.test(commit.message);
  core.info(`${commit.sha}: ${ok ? 'ok' : 'fail'}`);
  if (!ok)
    failed.push(commit.sha);
}

if (failed.len() != 0)
  core.setFailed(`Commits with bad messages: ${failed}`);

return;
