import datetime
from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_mrr_waterfall_response_200 import GetMRRWaterfallResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    start: datetime.date | Unset = UNSET,
    end: datetime.date | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_start: str | Unset = UNSET
    if not isinstance(start, Unset):
        json_start = start.isoformat()
    params["start"] = json_start

    json_end: str | Unset = UNSET
    if not isinstance(end, Unset):
        json_end = end.isoformat()
    params["end"] = json_end

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/analytics/mrr/waterfall",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetMRRWaterfallResponse200 | None:
    if response.status_code == 200:
        response_200 = GetMRRWaterfallResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | GetMRRWaterfallResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    start: datetime.date | Unset = UNSET,
    end: datetime.date | Unset = UNSET,
) -> Response[Error | GetMRRWaterfallResponse200]:
    """MRR waterfall

     MRR movement between two dates — new, expansion, contraction, churned, reactivation — plus Net and
    Gross Dollar Retention. Defaults to the trailing month. Movement is only accurate for periods after
    MRR snapshots began; has_start_history flags when the start is before that.

    Args:
        start (datetime.date | Unset):
        end (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetMRRWaterfallResponse200]
    """

    kwargs = _get_kwargs(
        start=start,
        end=end,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    start: datetime.date | Unset = UNSET,
    end: datetime.date | Unset = UNSET,
) -> Error | GetMRRWaterfallResponse200 | None:
    """MRR waterfall

     MRR movement between two dates — new, expansion, contraction, churned, reactivation — plus Net and
    Gross Dollar Retention. Defaults to the trailing month. Movement is only accurate for periods after
    MRR snapshots began; has_start_history flags when the start is before that.

    Args:
        start (datetime.date | Unset):
        end (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetMRRWaterfallResponse200
    """

    return sync_detailed(
        client=client,
        start=start,
        end=end,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    start: datetime.date | Unset = UNSET,
    end: datetime.date | Unset = UNSET,
) -> Response[Error | GetMRRWaterfallResponse200]:
    """MRR waterfall

     MRR movement between two dates — new, expansion, contraction, churned, reactivation — plus Net and
    Gross Dollar Retention. Defaults to the trailing month. Movement is only accurate for periods after
    MRR snapshots began; has_start_history flags when the start is before that.

    Args:
        start (datetime.date | Unset):
        end (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetMRRWaterfallResponse200]
    """

    kwargs = _get_kwargs(
        start=start,
        end=end,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    start: datetime.date | Unset = UNSET,
    end: datetime.date | Unset = UNSET,
) -> Error | GetMRRWaterfallResponse200 | None:
    """MRR waterfall

     MRR movement between two dates — new, expansion, contraction, churned, reactivation — plus Net and
    Gross Dollar Retention. Defaults to the trailing month. Movement is only accurate for periods after
    MRR snapshots began; has_start_history flags when the start is before that.

    Args:
        start (datetime.date | Unset):
        end (datetime.date | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetMRRWaterfallResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            start=start,
            end=end,
        )
    ).parsed
