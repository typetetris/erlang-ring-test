-module(ringtest).
-export([measured_createring/1,measured_send_and_wait/2, measure_ring/2, measure_ring/0, measure_ring_cli/1]).

measured(Fun) ->
        statistics(runtime),
        statistics(wall_clock),

        R = Fun(),

        {_, Time1} = statistics(runtime),
        {_, Time2} = statistics(wall_clock),
        {R, Time1, Time2}
        .

createringelem(0, HeadPid, RingRef) -> HeadPid ! {RingRef, done}, HeadPid;
createringelem(N, HeadPid, RingRef) -> spawn(
                          fun() ->
                                          Next = createringelem(N-1,HeadPid,RingRef),
                                          normalelemloop(Next,RingRef)
                          end)
                        .

normalelemloop(Next,RingRef) ->
        receive
                {RingRef, msg, die} = Z -> Next ! Z;
                {RingRef, msg, _} = Z ->
                        Next ! Z,
                        normalelemloop(Next,RingRef);
                Other ->
                        io:format("Got bogus message: ~p~n, dying now.", [Other])

        end
        .

createring(0) -> io:format("Ring with 0 elements? Feeling funny today, eh?~n"), error;
createring(N) -> RingRef = erlang:make_ref(),
                 CreatorPid = self(),
                 HeadPid = spawn(
                             fun () ->
                                             Next = createringelem(N-1, self(), RingRef),
                                             ringheadloop(Next, RingRef, CreatorPid)
                             end
                            ),
                 receive
                         {HeadPid, created} -> void
                 end,
                 HeadPid
                 .

ringheadloop(Next, RingRef, CreatorPid) ->
        receive
                {RingRef, msg, die} -> CreatorPid ! {self(), done};
                {RingRef, done} -> CreatorPid ! {self(), created}, ringheadloop(Next, RingRef, CreatorPid);
                {RingRef, msg, N} when is_integer(N) andalso N > 0 -> Next ! {RingRef, msg, N-1}, ringheadloop(Next,RingRef,CreatorPid);
                {RingRef, msg, _} -> CreatorPid ! {self(), done}, ringheadloop(Next,RingRef,CreatorPid);
                Other -> Next ! {RingRef, msg, Other}, ringheadloop(Next,RingRef,CreatorPid)
        end
        .

measured_createring(N) -> 
        {HeadPid, Runtime, Clocktime} = measured(fun() -> createring(N) end),
        io:format("ring creation took: ~p (~p) milliseconds~n", [Runtime, Clocktime]),
        HeadPid.

measured_send_and_wait(HeadPid, Msg) ->
        {_, Runtime, Clocktime} = measured(fun() ->
                                                           HeadPid ! Msg,
                                                           receive
                                                                   {HeadPid, done} -> void
                                                           end
                                           end
                                          ),
        io:format("sending the message around the ring"),
        if is_integer(Msg) -> io:format(" ~p times", [Msg]); true -> void end,
        io:format(" took ~p (~p) milliseconds~n", [Runtime, Clocktime])
        .

measure_ring(N,M) ->
        Ring = measured_createring(N),
        measured_send_and_wait(Ring,M)
        .

measure_ring_cli([Na,Ma]) ->
        N = list_to_integer(atom_to_list(Na)),
        M = list_to_integer(atom_to_list(Ma)),
        measure_ring(N,M)
        .


measure_ring() -> measure_ring(10000,1000).
