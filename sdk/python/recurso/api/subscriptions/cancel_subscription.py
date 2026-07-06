from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.cancel_subscription_request import CancelSubscriptionRequest
from ...models.cancel_subscription_response import CancelSubscriptionResponse
from ...models.error import Error
from ...types import Response


def _get_kwargs(
    id: UUID,
    *,
    body: CancelSubscriptionRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions/{id}/cancel".format(
            id=quote(str(id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CancelSubscriptionResponse | Error | None:
    if response.status_code == 200:
        response_200 = CancelSubscriptionResponse.from_dict(response.json())

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
) -> Response[CancelSubscriptionResponse | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CancelSubscriptionRequest,
) -> Response[CancelSubscriptionResponse | Error]:
    """Cancel a subscription

     Defaults to cancellation at period end unless `immediately` is true.

    Args:
        id (UUID):
        body (CancelSubscriptionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CancelSubscriptionResponse | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CancelSubscriptionRequest,
) -> CancelSubscriptionResponse | Error | None:
    """Cancel a subscription

     Defaults to cancellation at period end unless `immediately` is true.

    Args:
        id (UUID):
        body (CancelSubscriptionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CancelSubscriptionResponse | Error
    """

    return sync_detailed(
        id=id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CancelSubscriptionRequest,
) -> Response[CancelSubscriptionResponse | Error]:
    """Cancel a subscription

     Defaults to cancellation at period end unless `immediately` is true.

    Args:
        id (UUID):
        body (CancelSubscriptionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CancelSubscriptionResponse | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CancelSubscriptionRequest,
) -> CancelSubscriptionResponse | Error | None:
    """Cancel a subscription

     Defaults to cancellation at period end unless `immediately` is true.

    Args:
        id (UUID):
        body (CancelSubscriptionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CancelSubscriptionResponse | Error
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            body=body,
        )
    ).parsed
