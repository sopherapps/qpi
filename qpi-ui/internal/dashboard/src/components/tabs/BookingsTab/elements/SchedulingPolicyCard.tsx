export function SchedulingPolicyCard() {
  return (
    <div className="bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 p-6 rounded-lg space-y-4">
      <h3 className="text-base font-bold text-gray-900 dark:text-white font-geist">
        Scheduling Policy
      </h3>
      <div className="space-y-3 text-xs text-gray-500 dark:text-zinc-400 leading-relaxed">
        <p>
          <strong>Dedicated Window:</strong> When you book a slot, you have
          exclusive priority to submit jobs to the active QPUs.
        </p>
        <p>
          <strong>Opportunistic FIFO:</strong> If the slot owner is idle (has
          not submitted a job) for more than 5 seconds, other pending jobs from
          the queue will automatically run to optimize machine usage.
        </p>
        <p>
          <strong>Releasing Slots:</strong> You can cancel/delete your booked
          slots at any time prior to the start time.
        </p>
      </div>
    </div>
  );
}
